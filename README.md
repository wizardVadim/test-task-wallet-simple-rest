# test-task-wallet-simple-rest

## Start

```bash
docker compose --env-file config.env up -d
```
or
```bash
make docker-start
```

## Stop

```bash
docker compose down
```
or
```bash
make docker-stop
```

## Tests
```bash
make local-test-go
```

## Local run
```bash
make local-run-go
```


## Repository integration tests

Requires Go, Docker Compose with `up --wait` support, and a running local Docker daemon:

```bash
make integration-test
```

The command uses `docker-compose.test.yaml` to start a temporary PostgreSQL container on an available local port,
waits for it to become ready, and runs the repository integration tests.
The test containers, network, and volumes are removed after success, failure, or interruption.
Each test applies the project migration in a separate schema and cleans it up afterward.
These tests do not use `config.env` or the development database.

`make local-test-go` skips these tests unless `TEST_DATABASE_URL` is set.

## Load testing

The API was tested with Vegeta against a single wallet.

Test parameters:

- Rate: 1000 requests/sec
- Duration: 30 seconds
- Total requests: 30,000
- Success rate: 100%
- HTTP 200 responses: 30,000
- p95 latency: ~3.24s
- p99 latency: ~3.62s

All requests were performed against the same wallet to verify
correctness under concurrent balance updates.

After 30,000 successful DEPOSIT operations with amount=1,
the wallet balance increased exactly by 30,000, confirming
that no updates were lost.