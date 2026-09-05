.PHONY: local-run-go
.PHONY: local-test-go

local-run-go:
	@set -a && \
	. ./config.env && \
	set +a && \
	go run ./cmd


local-test-go:
	@go test ./cmd/... ./internal/...
