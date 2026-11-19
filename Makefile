.PHONY: env test test-integration

env:
	@test -e .env || cp .env.example .env

test:
	go test -race ./...

test-integration:
	go test -race -tags=integration ./...
