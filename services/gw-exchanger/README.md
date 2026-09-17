# gw-exchanger

A gRPC service providing USD, RUB and EUR exchange rates from PostgreSQL.
Run the commands below from the repository root.

## Run

Create `config.env` from `example_config.env` on first setup. To build and start
exchanger with its database and migrations:

```bash
docker compose --env-file config.env up -d --build exchanger
docker compose --env-file config.env logs migrate-exchanger exchanger
```

For local development:

```bash
make db-exchanger
make local-run-exchanger
```

The local target runs `POSTGRES_HOST_EXCHANGER=localhost go run
./services/gw-exchanger/cmd -c config.env`. Stop an exchanger container using the
same listener port before running locally, or override `GRPC_PORT`.

## Configuration

| Variable | Meaning | Example |
|---|---|---|
| `POSTGRES_HOST_EXCHANGER` | Database hostname | `db-exchanger` |
| `POSTGRES_PORT_EXCHANGER` | Database port; published host port in root config | `5431` |
| `POSTGRES_USER_EXCHANGER` | Database user | `test_user` |
| `POSTGRES_PASSWORD_EXCHANGER` | Local development database password | `test_pass` |
| `POSTGRES_NAME_EXCHANGER` | Database name | `exchanger` |
| `GRPC_ADDR` | Local listener address | `0.0.0.0` |
| `GRPC_PORT` | Listener port | `50051` |
| `GRPC_OUT_PORT` | Host port published by Compose | `50051` |
| `LOG_LEVEL_EXCHANGER` | Optional DEBUG, INFO, WARN or ERROR | `INFO` |

Compose connects exchanger to `db-exchanger:5432` and binds its listener to
`0.0.0.0`. Locally, connect to PostgreSQL at `localhost:5431` with the example settings.
`GRPC_OUT_PORT` is used by Compose, not read by the application.

With `-c config.env`, configuration comes from the file with explicit environment
variables taking precedence. Without `-c`, only environment variables are read.
Data persists in `out/exchanger_pg_data`.

## Rates and gRPC

The contract is [exchange.proto](../../contracts/exchange/exchange.proto).
`units_per_usd` means units of the row's currency per 1 USD. Migrations seed these
example rates: USD = 1, RUB = 90.1, EUR = 0.87. They are not live market rates.

`GetExchangeRates` returns `base_currency = "USD"` and a map of rates.
`GetExchangeRateForCurrency` returns the requested pair; its rate is
`units_per_usd(to) / units_per_usd(from)`. Valid identical currencies return 1.
Input currency codes are normalized to uppercase. Values use float32 precision;
the service supplies rates and does not move money.

With grpcurl installed, supply the proto explicitly (server reflection is not enabled):

```bash
grpcurl -plaintext -import-path contracts/exchange -proto exchange.proto \
  -d '{}' localhost:50051 exchange.ExchangeService/GetExchangeRates

grpcurl -plaintext -import-path contracts/exchange -proto exchange.proto \
  -d '{"from_currency":"RUB","to_currency":"EUR"}' \
  localhost:50051 exchange.ExchangeService/GetExchangeRateForCurrency
```

| gRPC status | Meaning |
|---|---|
| `InvalidArgument` | Unsupported or empty request currency |
| `NotFound` | A requested base rate is absent |
| `Canceled` / `DeadlineExceeded` | Request canceled or deadline expired |
| `Internal` | Database, stored-data or other internal failure |

## Logs and shutdown

JSON logs include `service=gw-exchanger`. RPC completion logs record the method,
status and duration without request bodies or credentials. Startup and shutdown
are logged. SIGINT/SIGTERM starts graceful shutdown; after 5 seconds remaining
RPCs are forcibly stopped.

## Tests

```bash
make test-exchanger
```

Tests cover domain validation, cross rates, repository error handling, configuration,
CLI flags, gRPC responses and logging. The transport test uses a gRPC client/server
connected in memory. Repository tests currently use test doubles; a PostgreSQL
integration suite for exchanger is still pending.
