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
