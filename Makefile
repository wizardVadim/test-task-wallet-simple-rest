.PHONY: local-run-go
.PHONY: local-test-go
.PHONY: docker-start

local-run-go:
	@set -a && \
	. ./config.env && \
	set +a && \
	POSTGRES_HOST=localhost go run ./cmd


local-test-go:
	@go test ./cmd/... ./internal/...

docker-start:
	@docker compose --env-file config.env up -d

docker-stop:
	@docker compose down