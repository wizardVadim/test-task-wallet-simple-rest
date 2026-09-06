# test-task-wallet-simple-rest

A wallet REST API built with Go and PostgreSQL. Deposits and withdrawals use
atomic SQL updates, with balance checks in the same statement to prevent lost
updates, negative balances, and integer overflow under concurrent requests.

## Quick start

Requires Docker and Docker Compose. Run commands from the repository root:

```bash
cp example_config.env config.env
docker compose --env-file config.env up --build -d
```

Compose starts PostgreSQL, applies migrations using the migrate container,
and starts the API after migrations succeed. No local Go or migrate installation
is required for this workflow. The default API URL is `http://localhost:8080`.

Inspect startup status and logs:

```bash
docker compose --env-file config.env ps -a
docker compose --env-file config.env logs migrate wallet
```

Stop the application:

```bash
docker compose --env-file config.env down
```

PostgreSQL data persists in `out/pg_data`. Local configuration and database data
are excluded from Git. Database initialization settings apply to a new data directory;
changing credentials in the env file does not update an existing database user.

Make shortcuts:

```bash
make docker-start
make docker-stop
make docker-rebuild
```

Use `docker-rebuild` after changing application code; restarting an existing
container does not rebuild its image.

## Configuration

Copy `example_config.env` to `config.env` before starting the application.
Compose passes the file's variables to the API, which reads them from its environment.

| Variable | Purpose | Example |
|---|---|---|
| `POSTGRES_USER` | Database user | `test_user` |
| `POSTGRES_PASSWORD` | Database password | `test_pass` |
| `POSTGRES_HOST` | Database host inside Compose | `db` |
| `POSTGRES_NAME` | Database name | `bank` |
| `POSTGRES_PORT` | Published database port for local connections | `5432` |
| `HTTP_ADDR` | API listening port, without a colon | `8080` |
| `HTTP_PORT` | Published API port on the host | `8080` |

The API container overrides `POSTGRES_PORT` to `5432`, the PostgreSQL port inside
Compose. HTTP ports are mapped as `HTTP_PORT:HTTP_ADDR`.

## API

Amounts and balances are integers in minor monetary units: `100` represents
1.00 monetary units. The API has no currency field and performs no conversion.
Operation amounts must be positive and fit in `int64`. Balances range from
zero to `9223372036854775807`; overdrafts are not supported.

### Create a wallet

`POST /api/v1/wallets` creates a wallet with a server-generated UUID and zero balance.
No request body is required.

```bash
curl -i -X POST http://localhost:8080/api/v1/wallets
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
curl -i "http://localhost:8080/api/v1/wallets/$WALLET_ID"
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
  -H 'Content-Type: application/json' \
  -d "{\"walletId\":\"$WALLET_ID\",\"operationType\":\"DEPOSIT\",\"amount\":1000}"
```

Withdraw:

```bash
curl -i -X POST http://localhost:8080/api/v1/wallet \
  -H 'Content-Type: application/json' \
  -d "{\"walletId\":\"$WALLET_ID\",\"operationType\":\"WITHDRAW\",\"amount\":1000}"
```

Both operations return `200 OK` with an empty body after a successful update.
Operation types are case-insensitive. Withdrawing the entire balance is allowed.
Each accepted POST is a separate operation; automatic retries can apply it again.

## Error responses

Handler errors use this JSON envelope:

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

Requires the Go version specified in `go.mod` and Make. Start the database and
apply migrations, then run the API locally:

```bash
docker compose --env-file config.env up -d db migrate
docker compose --env-file config.env wait migrate
make local-run-go
```

Ensure migrations succeeded before running the API. Stop any existing `wallet`
container if it occupies the local API port. The Make target loads `config.env`
and overrides `POSTGRES_HOST` to `localhost`. Locally the API listens on `HTTP_ADDR`.

## Tests

Run unit tests without PostgreSQL:

```bash
make local-test-go
```

Repository integration tests are skipped unless `TEST_DATABASE_URL` is set.

Run integration tests with a temporary PostgreSQL instance:

```bash
make integration-test
```

Requires Go, Make, Docker Compose with `up --wait` support, and a running local
Docker daemon. The command uses `docker-compose.test.yaml`, selects an available
local port, waits for database readiness, and runs the repository tests.
Containers, network, and volumes are removed after success, failure, or interruption.
Each test applies the project migration in a separate schema and cleans it up afterward.
These tests do not use `config.env` or the development database.

Coverage includes domain validation, configuration, service error propagation,
HTTP responses, atomic balance updates, balance limits, and concurrent deposits.

## Load testing

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

Run from the repository root with Vegeta installed and the API running.
Set the wallet UUID in the request body files and adjust the target URL if needed:

```bash
vegeta attack -targets=./loadtests/vegeta_targets_add_balance.txt -rate=1000 -duration=60s | vegeta report
```

For withdrawals, prepare a starting balance of at least 60,000:

```bash
vegeta attack -targets=./loadtests/vegeta_targets_minus_balance.txt -rate=1000 -duration=60s | vegeta report
```

The `make load-test-add-balance`, `make load-test-minus-balance`, and
`make load-test-get-balance` shortcuts currently run for **30 seconds**.
Use the explicit 60-second commands above to reproduce the duration in the results table.
After each run, verify the final balance against the starting balance and successful operations.
