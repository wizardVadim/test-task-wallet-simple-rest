# Wallet services

Go monorepo for a wallet and its supporting services. Currently implemented:
[`gw-currency-wallet`](services/gw-currency-wallet/README.md), a REST API with
PostgreSQL. `gw-exchanger` is in development: its domain types, tests, gRPC contract
and PostgreSQL repository are present; its application and gRPC server are not implemented yet.
Wallet currency exchange, authentication, notifications and analytics are planned.

## Structure

```text
services/gw-currency-wallet/  Application, Go module, Dockerfile, migrations and tests
services/gw-exchanger/       Exchange domain, Go module, migrations and tests
contracts/                  Shared Go module with protobuf definitions and generated code
go.work                     Local Go workspace
docker-compose.yaml         Wallet, two PostgreSQL instances and their migrations
docker-compose.test.yaml    Temporary PostgreSQL for integration tests
Makefile                    Commands run from the repository root
```

## Run

Run all commands below from the repository root. Requires Docker and Docker Compose.
Create local configuration on first setup (keep an existing config.env):

```bash
cp example_config.env config.env
```

Build and start:

```bash
docker compose --env-file config.env up --build -d
docker compose --env-file config.env ps -a
```

The default API address is `http://localhost:8080`. PostgreSQL data persists in
`out/wallet_pg_data` for wallet and `out/exchanger_pg_data` for exchanger.
Compose applies `migrate-wallet` before starting the wallet; `migrate-exchanger`
applies the exchanger schema to its separate database.
See the [service README](services/gw-currency-wallet/README.md) for configuration,
API requests, errors and load testing.

```bash
make docker-rebuild
make docker-stop
```

`make docker-rebuild` rebuilds the wallet application. `make docker-stop` stops
all services in the main Compose stack.

## Exchanger database

The exchanger database can be started with its migrations independently:

```bash
docker compose --env-file config.env up -d db-exchanger migrate-exchanger
docker compose --env-file config.env wait migrate-exchanger
docker compose --env-file config.env logs migrate-exchanger
```

`db-exchanger` should become healthy and `migrate-exchanger` should exit with code 0.
The migrations create `exchange_rates` and seed example rates relative to USD:
USD = 1, RUB = 90.1, EUR = 0.87. These are local test values, not live market rates.

The root `config.env` and `example_config.env` contain these exchanger settings:

| Variable | Purpose | Example |
|---|---|---|
| `POSTGRES_USER_EXCHANGER` | Database user | `test_user` |
| `POSTGRES_PASSWORD_EXCHANGER` | Local development database password | `test_pass` |
| `POSTGRES_HOST_EXCHANGER` | Database hostname inside Compose | `db-exchanger` |
| `POSTGRES_NAME_EXCHANGER` | Database name | `exchanger` |
| `POSTGRES_PORT_EXCHANGER` | Published database port on the host | `5431` |

From the host, connect to `localhost:5431` with the example settings. Inside Compose,
use `db-exchanger:5432`; the migration service uses this internal address.
`POSTGRES_HOST_EXCHANGER` is reserved for the future application configuration;
the migration service currently uses the hostname directly in its connection URL.
Database initialization settings take effect only for an empty data directory.

## Local development and tests

Local Go commands use the root `go.work`; use a compatible Go toolchain.
The Docker build builds the wallet module independently of the workspace.

```bash
make local-test-go
make integration-test
```

The Make test targets above cover wallet only. To run unit tests across both services
and check compilation of the shared contracts:

```bash
go test ./services/gw-currency-wallet/... ./services/gw-exchanger/... ./contracts/...
```

Integration tests require Docker. They create a temporary PostgreSQL instance and
clean it up afterward; they do not use the development database.

To run the API locally, first stop any wallet container using its HTTP port,
then start PostgreSQL and apply migrations:

```bash
docker compose --env-file config.env up -d db-currency-wallet migrate-wallet
docker compose --env-file config.env wait migrate-wallet
make local-run-go
```

`make local-run-go` reads the root `config.env` and uses `localhost` for PostgreSQL.
