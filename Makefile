.PHONY: build test test-unit test-integration lint fmt install-hooks

build:
	go build -o gocker .

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

test-unit:
	go test ./internal/... -count=1

test-integration: build
	go test ./tests/integration/... -count=1

test: test-unit test-integration

install-hooks:
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "pre-commit hook installed"
