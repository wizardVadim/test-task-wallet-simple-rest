# Wallet services

Go monorepo with two services:

- [gw-currency-wallet](services/gw-currency-wallet/README.md): wallet REST API with PostgreSQL.
- [gw-exchanger](services/gw-exchanger/README.md): gRPC exchange rates from a separate PostgreSQL database.

Wallet currently supports anonymous wallets, balance reads, deposits and withdrawals.
Connecting wallet to exchanger, authentication, multicurrency balances, Kafka,
notifications and analytics are still planned.

## Structure

```text
services/gw-currency-wallet/  REST application, migrations and tests
services/gw-exchanger/        gRPC application, migrations and tests
contracts/                   Protobuf definitions and generated Go code
go.work                      Local workspace for all three Go modules
docker-compose.yaml          Both applications, two databases and migration jobs
docker-compose.test.yaml     Temporary database for wallet integration tests
Makefile                     Commands run from the repository root
```

## Setup and Docker

Run commands from the repository root. Docker workflows require Docker Compose;
local development also requires Make and a Go toolchain compatible with `go.work`.
Both Dockerfiles use `golang:1.27.1-bookworm`, matching the workspace and exchanger/contracts minimum version.
On first setup, create the configuration (keep an existing `config.env`):

```bash
cp example_config.env config.env
make docker-rebuild
docker compose --env-file config.env ps -a
docker compose --env-file config.env logs wallet exchanger
```

Each application starts after its database is healthy and its migration job succeeds.
Settings in `example_config.env` use these addresses:

| Component | From your computer | Inside Compose |
|---|---|---|
| Wallet REST API | `http://localhost:8080` | `wallet:8080` |
| Exchanger gRPC | `localhost:50051` | `exchanger:50051` |
| Wallet PostgreSQL | `localhost:5432` | `db-currency-wallet:5432` |
| Exchanger PostgreSQL | `localhost:5431` | `db-exchanger:5432` |

`HTTP_PORT` and `GRPC_PORT` control application listeners; `HTTP_OUT_PORT` and
`GRPC_OUT_PORT` control published host ports. Both database containers listen on
5432; `POSTGRES_PORT` and `POSTGRES_PORT_EXCHANGER` select their host ports.
Compose overrides the applications' database ports to 5432.

Database files persist in `out/wallet_pg_data` and `out/exchanger_pg_data`.
Stopping Compose preserves them. PostgreSQL initialization settings apply only
to an empty data directory; editing credentials does not change an existing user.

## Make commands

| Command | Action |
|---|---|
| `make docker-start` | Start the full stack using existing images; build missing images |
| `make docker-rebuild` | Build and start both applications and their dependencies |
| `make docker-stop` | Stop and remove Compose containers and network; preserve database files |
| `make db-wallet` | Start wallet PostgreSQL and apply wallet migrations |
| `make db-exchanger` | Start exchanger PostgreSQL and apply exchanger migrations |
| `make local-run-wallet` | Run wallet locally with `-c config.env` and database host `localhost` |
| `make local-run-exchanger` | Run exchanger locally with `-c config.env` and database host `localhost` |
| `make local-run-go` | Alias for `local-run-wallet` |
| `make local-test-go` | Test both services and compile shared contracts |
| `make test-wallet` | Run wallet tests |
| `make test-exchanger` | Run exchanger tests |
| `make integration-test` | Run both repository suites against temporary PostgreSQL |

`docker-start` does not rebuild existing images after source changes; use
`docker-rebuild`. For one application only, use `docker compose --env-file
config.env up -d --build wallet` or the same command with `exchanger`.

## Local run and configuration

Stop the corresponding application container if it occupies the local listener port.
In separate terminals:

```bash
make db-wallet
make local-run-wallet
```

```bash
make db-exchanger
make local-run-exchanger
```

Both applications support `-c config.env`. Explicit environment variables override
file values, including empty values. Without `-c`, applications read only the
environment; Compose uses this mode through `env_file`.

The local Make targets override the database hostname, keeping the published database
port from configuration. Local listeners use `HTTP_PORT` / `GRPC_PORT`, not the
`*_OUT_PORT` settings. For example, `GRPC_PORT=50052 make local-run-exchanger`
selects a different local port.

JSON logs go to stdout; startup failures go to stderr. `LOG_LEVEL_WALLET` and
`LOG_LEVEL_EXCHANGER` accept DEBUG, INFO, WARN or ERROR and default to INFO.
See the service READMEs for request examples and details.

## Tests

```bash
make local-test-go
make integration-test
```

Without `TEST_DATABASE_URL`, repository integration tests for both services are skipped.
`make integration-test` creates a temporary PostgreSQL instance, runs those tests,
and removes its containers, network and volumes afterward. It does not use the
development databases or `config.env`.

Exchanger has domain, service, configuration, repository unit tests and gRPC tests
using an in-memory connection. Its PostgreSQL integration tests cover seeded rates,
empty results, concurrent reads, invalid stored currencies, database constraints
and migration rollback/reapplication. Each test uses its own schema.

Wallet load-test targets remain `load-test-add-balance`, `load-test-minus-balance`
and `load-test-get-balance`; they require Vegeta and run for 30 seconds.
