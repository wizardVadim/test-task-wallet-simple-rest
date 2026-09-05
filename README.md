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
