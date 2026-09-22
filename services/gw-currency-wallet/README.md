# gw-currency-wallet

A wallet REST API built with Go and PostgreSQL. Deposits and withdrawals use
atomic SQL updates, with balance checks in the same statement to prevent lost
updates, negative balances, and integer overflow under concurrent requests.

Unless stated otherwise, run commands from the repository root (two levels above this directory).
See the [repository README](../../README.md) for the workspace layout.

## Quick start

Requires Docker and Docker Compose. Run commands from the repository root:

```bash
cp example_config.env config.env
# Set your own JWT_SECRET_KEY in config.env before starting.
docker compose --env-file config.env up --build -d
```

Compose starts `db-currency-wallet`, applies migrations using `migrate-wallet`,
and starts the API after migrations succeed. No local Go or migrate installation
is required for this workflow. The full Compose stack also starts the exchanger database
and its migrations, as well as the exchanger application. The default API URL is `http://localhost:8080`.

Inspect startup status and logs:

```bash
docker compose --env-file config.env ps -a
docker compose --env-file config.env logs migrate-wallet wallet
```

Stop the application:

```bash
docker compose --env-file config.env down
```

PostgreSQL data persists in `out/wallet_pg_data`. Local configuration and database data
are excluded from Git. Database initialization settings apply to a new data directory;
changing credentials in the env file does not update an existing database user.

Make shortcuts:

```bash
make docker-start
make docker-stop
make docker-rebuild
```

`docker-rebuild` rebuilds both applications. To rebuild wallet only, run
`docker compose --env-file config.env up -d --build wallet`. Restarting an existing
container does not rebuild its image.

## Configuration

Set `JWT_SECRET_KEY` to a random secret of at least 32 bytes; `openssl rand -hex 32`
can generate one. Replace the placeholder in `config.env` and keep this file out
of Git. Changing the signing key invalidates previously issued tokens.


Copy `example_config.env` to `config.env` before starting the application.
Compose passes the file's variables to the API, which reads them from its environment.
For direct local execution, `-c config.env` reads the file without exporting it;
explicit environment variables override file values.

| Variable | Purpose | Example |
|---|---|---|
| `POSTGRES_USER` | Database user | `test_user` |
| `POSTGRES_PASSWORD` | Database password | `test_pass` |
| `POSTGRES_HOST` | Database host inside Compose | `db-currency-wallet` |
| `POSTGRES_NAME` | Database name | `bank` |
| `POSTGRES_PORT` | Published database port for local connections | `5432` |
| `HTTP_PORT` | API listening port, without a colon | `8080` |
| `HTTP_OUT_PORT` | Published API port on the host | `8080` |
| `JWT_SECRET_KEY` | Required HS256 signing secret, at least 32 bytes | Your own random secret |
| `JWT_TTL` | Required positive integer lifetime in hours | `24` |
| `LOG_LEVEL_WALLET` | Optional JSON log level: DEBUG, INFO, WARN, ERROR | `INFO` |
| `MAX_DB_CONNECTIONS` | Maximum connections in the API database pool | `4` |
| `MIN_DB_CONNECTIONS` | Minimum connections maintained by the pool | `1` |
| `READ_HEADER_TIMEOUT` | HTTP request header read timeout, in seconds | `5` |
| `READ_TIMEOUT` | Entire HTTP request read timeout, in seconds | `5` |
| `WRITE_TIMEOUT` | HTTP response write timeout, in seconds | `20` |
| `IDLE_TIMEOUT` | HTTP keep-alive idle timeout, in seconds | `120` |

The API container overrides `POSTGRES_PORT` to `5432`, the PostgreSQL port inside
Compose. HTTP ports are mapped as `HTTP_OUT_PORT:HTTP_PORT`.

The application requires all settings above except `LOG_LEVEL_WALLET` and
`HTTP_OUT_PORT`; the latter is used only by Compose. Pool limits and timeouts must be
integers. `MAX_DB_CONNECTIONS` must be between 1 and 2147483647;
`MIN_DB_CONNECTIONS` must be between 0 and `MAX_DB_CONNECTIONS`, inclusive.
Timeouts must be positive and fit in a Go duration when converted from seconds
(at most 9223372036 seconds). These are validation limits, not recommended tuning values.
`JWT_TTL` accepts integer hours from 1 to 2562047; values such as `24h` or `1.5`
are rejected. Its conversion to a duration is checked for overflow.
Invalid pool or timeout settings cause startup to fail with the variable name in the error.

To compare pool sizes, change `MAX_DB_CONNECTIONS` in `config.env`, keeping
`MIN_DB_CONNECTIONS` no greater than the maximum, and recreate the API container:

```bash
docker compose --env-file config.env up -d --no-deps --force-recreate wallet
```

An image rebuild is not needed for environment-only changes once the image includes
support for these settings. A plain container restart does not reload `config.env`.
Run identical load scenarios for each pool size and compare throughput, latency,
errors, and final balances. Increasing the pool size does not guarantee higher throughput
when all updates target the same wallet. `WRITE_TIMEOUT` controls response writes;
it does not set a database query timeout.

## Authentication

`POST /api/v1/register` and `POST /api/v1/login` are public. All three wallet
routes below require `Authorization: Bearer <token>`. Missing, invalid or expired
tokens receive `401 Unauthorized` with an empty body.

Wallets currently have no owner relationship. A valid token permits access to any
wallet ID; restricting users to their own balances is planned with multicurrency wallets.
Registration creates a user only; it does not create a wallet.

### Register

```bash
curl -i -X POST http://localhost:8080/api/v1/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"ivan","email":"ivan@example.com","password":"example-password"}'
```

Success: `201 Created`, `{"message":"User registered successfully"}`.
Username is trimmed and lowercased. Email syntax is checked and email uniqueness
is case-insensitive. Passwords must be nonblank and at most 72 bytes, not characters;
spaces in an otherwise valid password are preserved. Passwords are stored as bcrypt hashes.
Duplicate username or email: `400`, `{"error":"Username or email already exists"}`.
Other invalid registration fields also return `400` with an `error` string.

### Log in

```bash
curl -i -X POST http://localhost:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"ivan","password":"example-password"}'
```

Success: `200 OK`, `{"token":"<JWT>"}`. Copy the returned token for subsequent examples:

```bash
TOKEN='<JWT returned by login>'
```

Invalid credentials or invalid password: `401`,
`{"error":"Invalid username or password"}`. Unexpected internal errors return
`500`, `{"error":"internal server error"}`. Both auth endpoints accept one JSON
value with a 4096-byte limit: malformed input returns `400`, oversized input `413`.
Auth errors use `{"error":"..."}`, unlike the wallet error envelope below.

JWTs use HS256 with `sub` (user ID), `iat` (issued at), and `exp` (expiry).
There is no refresh-token or individual-token revocation endpoint; log in again
when the token expires. Middleware verifies the signature, algorithm and expiry,
then passes a nonzero UUID user ID through the request context.

## API

Amounts and balances are integers in minor monetary units: `100` represents
1.00 monetary units. The API has no currency field and performs no conversion.
Operation amounts must be positive and fit in `int64`. Balances range from
zero to `9223372036854775807`; overdrafts are not supported.

### Create a wallet

`POST /api/v1/wallets` creates a wallet with a server-generated UUID and zero balance.
No request body is required.

```bash
curl -i -X POST http://localhost:8080/api/v1/wallets \
  -H "Authorization: Bearer $TOKEN"
```

Returns `201 Created` and a `Location` header pointing to the wallet's GET endpoint:

```json
{
  "payload": {
    "walletId": "63e8c3d9-e907-4739-b2d0-6c751a887b4a",
    "balance": 0
  }
}
```

Use the returned UUID in subsequent requests:

```bash
WALLET_ID='63e8c3d9-e907-4739-b2d0-6c751a887b4a'
```

### Read the balance

`GET /api/v1/wallets/{wallet_uuid}` returns `200 OK`:

```bash
curl -i "http://localhost:8080/api/v1/wallets/$WALLET_ID" \
  -H "Authorization: Bearer $TOKEN"
```

```json
{
  "payload": {
    "balance": 1000
  }
}
```

### Deposit or withdraw

`POST /api/v1/wallet` accepts one JSON object, up to 4096 bytes:

```json
{
  "walletId": "63e8c3d9-e907-4739-b2d0-6c751a887b4a",
  "operationType": "DEPOSIT",
  "amount": 1000
}
```

Deposit:

```bash
curl -i -X POST http://localhost:8080/api/v1/wallet \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"walletId\":\"$WALLET_ID\",\"operationType\":\"DEPOSIT\",\"amount\":1000}"
```

Withdraw:

```bash
curl -i -X POST http://localhost:8080/api/v1/wallet \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"walletId\":\"$WALLET_ID\",\"operationType\":\"WITHDRAW\",\"amount\":1000}"
```

Both operations return `200 OK` with an empty body after a successful update.
Operation types are case-insensitive. Withdrawing the entire balance is allowed.
Each accepted POST is a separate operation; automatic retries can apply it again.

## Error responses

Wallet handler errors use this JSON envelope (authentication failures are described above):

```json
{
  "payload": null,
  "error": {
    "message": "wallet not found"
  }
}
```

| HTTP status | Message | Description |
|---|---|---|
| `400 Bad Request` | `invalid wallet id` | Invalid or nil wallet UUID |
| `400 Bad Request` | `invalid request body` | Malformed JSON, invalid fields, operation, or amount |
| `413 Request Entity Too Large` | `request body too large` | Request body exceeds 4096 bytes |
| `404 Not Found` | `wallet not found` | Wallet does not exist |
| `409 Conflict` | `insufficient funds` | Withdrawal exceeds the balance |
| `422 Unprocessable Entity` | `balance overflow` | Deposit exceeds the maximum balance |
| `500 Internal Server Error` | `couldn't get wallet balance` | Unexpected balance read failure |
| `500 Internal Server Error` | `couldn't create a new wallet` | Unexpected wallet creation failure |
| `500 Internal Server Error` | `couldn't change wallet balance` | Unexpected operation failure |

Unknown routes and unsupported HTTP methods use standard `net/http` responses.

## Local run

Requires a Go toolchain compatible with the root `go.work` and this service’s `go.mod`, plus Make. Start the database and
apply migrations, then run the API locally:

```bash
make db-wallet
make local-run-wallet
```

Ensure migrations succeeded before running the API. Stop any existing `wallet`
container if it occupies the local API port. The Make target loads `config.env`
and overrides `POSTGRES_HOST` to `localhost`. Locally the API listens on `HTTP_PORT`.

Alternatively, run from the repository root with file configuration:

```bash
POSTGRES_HOST=localhost go run ./services/gw-currency-wallet/cmd -c config.env
```

Without `-c`, the application reads only environment variables. The existing
`make local-run-go` target remains supported.

## Logging

The application writes JSON logs with `service=gw-currency-wallet`. Set
`LOG_LEVEL_WALLET` to `DEBUG`, `INFO` (default), `WARN` or `ERROR`.
Logs cover startup, shutdown, HTTP outcomes and successful balance changes.
HTTP records include method, route pattern, status and duration; request bodies,
query strings and authorization headers are not included. Startup failures go to
stderr; regular application logs go to stdout.

## Tests

Run unit tests without PostgreSQL:

```bash
make test-wallet
```

Repository integration tests are skipped unless `TEST_DATABASE_URL` is set.

Run integration tests with a temporary PostgreSQL instance:

```bash
make integration-test
```

Requires Go, Make, Docker Compose with `up --wait` support, and a running local
Docker daemon. The target covers wallet, auth and exchanger repositories. The command uses `docker-compose.test.yaml`, selects an available
local port, waits for database readiness, and runs the repository tests.
Containers, network, and volumes are removed after success, failure, or interruption.
Each test applies the project migration in a separate schema and cleans it up afterward.
These tests do not use `config.env` or the development database.

Coverage includes registration/login, bcrypt, JWT validation, authentication middleware,
JWT configuration precedence and bounds, domain validation, service error propagation,
HTTP responses, atomic balance updates, balance limits, and concurrent deposits.

## Load testing

The results below predate JWT authentication and do not measure the current authenticated API.

Two local Vegeta runs were reported against a single wallet, each configured
for 1,000 requests per second over 60 seconds with an operation amount of 1.
The figures below are from those runs; hardware and container resource limits
were not recorded.

| Metric | Deposit run | Withdrawal run |
|---|---:|---:|
| Total requests | 60,000 | 60,000 |
| Request rate | 1,000.02/s | 1,000.01/s |
| Throughput, including final wait | 861.46/s | 874.47/s |
| Successful responses | 100% (60,000 HTTP 200) | 100% (60,000 HTTP 200) |
| Mean latency | 4.132s | 3.584s |
| p50 latency | 4.073s | 3.496s |
| p95 latency | 8.670s | 7.491s |
| p99 latency | 9.490s | 8.334s |
| Maximum latency | 9.876s | 8.869s |
| Total duration | 69.649s | 68.613s |
| Wait after request generation ended | 9.650s | 8.614s |

The reported balance after the deposit run was exactly 60,000. With an initial
balance of zero and amount=1, this matches all 60,000 successful deposits.

The withdrawal target references `vegeta-minus-balance-body.json`, which uses
WITHDRAW with amount=1. All 60,000 withdrawals returned HTTP 200. The starting
balance was 120,000 and the confirmed final balance was 60,000, matching the
60,000 successful withdrawals.

These runs show that all 60,000 submitted requests received HTTP 200 responses
at an offered rate of 1,000 requests/sec. Completion continued for roughly
8–10 seconds after request generation stopped. They do not establish sustained
1,000 requests/sec completion throughput or bounded latency over longer runs.

Run the direct Vegeta commands below from `services/gw-currency-wallet/` with Vegeta installed and the API running.
From the repository root, first run `cd services/gw-currency-wallet`.
Set the wallet UUID in the request body files and adjust the target URL if needed.
Add `Authorization: Bearer <token>` to each Vegeta target file, immediately after
the request line, using a token from login. This is also required by the Make
load-test targets. Do not commit real tokens. Without this header requests return 401:

```bash
vegeta attack -targets=./loadtests/vegeta_targets_add_balance.txt -rate=1000 -duration=60s | vegeta report
```

For withdrawals, prepare a starting balance of at least 60,000:

```bash
vegeta attack -targets=./loadtests/vegeta_targets_minus_balance.txt -rate=1000 -duration=60s | vegeta report
```

The `make load-test-add-balance`, `make load-test-minus-balance`, and
`make load-test-get-balance` shortcuts are run from the repository root and currently run for **30 seconds**.
Use the explicit 60-second commands above to reproduce the duration in the results table.
After each run, verify the final balance against the starting balance and successful operations.
