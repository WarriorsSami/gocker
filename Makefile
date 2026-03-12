.PHONY: build test test-unit test-integration test-integration-build test-integration-root test-integration-uts-root lint fmt install-hooks

INTEGRATION_TEST_BIN := tests/integration/integration.test

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

test-integration-build: build
	go test -c ./tests/integration -o $(INTEGRATION_TEST_BIN)

test-integration-root: test-integration-build
	@echo "Running integration tests as root (requires sudo access)"
	sudo ./$(INTEGRATION_TEST_BIN) -test.v

test: test-unit test-integration

test-root: test-unit test-integration-root

install-hooks:
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "pre-commit hook installed"
