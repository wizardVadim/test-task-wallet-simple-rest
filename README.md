# test-task-wallet-simple-rest

## API

#### GET /api/v1/wallets/{wallet_uuid}

Response body:

```json
{
    "payload": {
        "balance:" 1000
    }
}
```

#### POST /api/v1/wallets

Response body:

```json
{
    "payload": {
        "walletId": "63e8c3d9-e907-4739-b2d0-6c751a887b4a",
        "balance": 0
    }
}
```

#### POST /api/v1/wallet

Request body:

```json
{
    "walletId": "63e8c3d9-e907-4739-b2d0-6c751a887b4a",
    "operationType": "DEPOSIT", // WITHDRAW
    "amount": 1000
}
```

## Error responses

All API errors use the following response format:

```json
{
    "error": {
        "message": "wallet not found"
    }
}
```

The API may return the following errors:

| HTTP Status | Message | Description |
|---|---|---|
| `400 Bad Request` | `invalid wallet id` | Wallet ID is not a valid UUID |
| `400 Bad Request` | `invalid request body` | Request body is invalid |
| `400 Bad Request` | `request body too large` | Request body exceeds the allowed size |
| `404 Not Found` | `wallet not found` | Wallet does not exist |
| `409 Conflict` | `insufficient funds` | Withdrawal amount exceeds the current balance |
| `409 Conflict` | `balance overflow` | Deposit would overflow the wallet balance |
| `500 Internal Server Error` | `couldn't get wallet balance` | Internal error while retrieving the balance |
| `500 Internal Server Error` | `couldn't create a new wallet` | Internal error while creating a wallet |
| `500 Internal Server Error` | `couldn't change wallet balance` | Internal error while changing the balance |

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

## Rebuild

```bash
make docker-rebuild
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