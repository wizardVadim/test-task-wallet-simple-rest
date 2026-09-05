.PHONY: local-run-go
.PHONY: local-test-go
.PHONY: docker-start
.PHONY: docker-stop integration-test

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

TEST_COMPOSE = docker compose -f docker-compose.test.yaml

integration-test:
	@set -eu; \
	export COMPOSE_PROJECT_NAME=wallet-tests-$$$$; \
	trap '$(TEST_COMPOSE) down --volumes' EXIT; \
	trap 'exit 130' INT; \
	trap 'exit 143' TERM; \
	$(TEST_COMPOSE) up --wait --wait-timeout 60; \
	test_address=$$($(TEST_COMPOSE) port db 5432); \
	TEST_DATABASE_URL="postgres://test_user:test_password@$$test_address/wallet_test?sslmode=disable" \
		go test -count=1 -v ./internal/features/wallet/repository
