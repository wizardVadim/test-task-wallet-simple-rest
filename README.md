# Wallet services

Go monorepo for a wallet and its supporting services. Currently implemented:
[`gw-currency-wallet`](services/gw-currency-wallet/README.md), a REST API with
PostgreSQL. Currency exchange, authentication, notifications and analytics are planned.

## Structure

```text
services/gw-currency-wallet/  Application, Go module, Dockerfile, migrations and tests
go.work                     Local Go workspace
docker-compose.yaml         Wallet, PostgreSQL and migrations
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
`out/pg_data`. Compose applies migrations before starting the wallet.
See the [service README](services/gw-currency-wallet/README.md) for configuration,
API requests, errors and load testing.

```bash
make docker-rebuild
make docker-stop
```

## Local development and tests

Local Go commands use the root `go.work`; use a compatible Go toolchain.
The Docker build builds the wallet module independently of the workspace.

```bash
make local-test-go
make integration-test
```

Integration tests require Docker. They create a temporary PostgreSQL instance and
clean it up afterward; they do not use the development database.

To run the API locally, first stop any wallet container using its HTTP port,
then start PostgreSQL and apply migrations:

```bash
docker compose --env-file config.env up -d db migrate
docker compose --env-file config.env wait migrate
make local-run-go
```

`make local-run-go` reads the root `config.env` and uses `localhost` for PostgreSQL.
