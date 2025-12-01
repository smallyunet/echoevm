BINARY_NAME ?= echoevm
BIN_DIR ?= bin

GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION    ?= dev
LDFLAGS    := -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE) -X main.Version=$(VERSION)

.PHONY: install build run test coverage clean help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

install: ## Install the echoevm binary to GOPATH/bin
	go install -ldflags "$(LDFLAGS)" ./cmd/echoevm

$(BIN_DIR)/$(BINARY_NAME):
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/echoevm

build: $(BIN_DIR)/$(BINARY_NAME) ## Build the echoevm binary

run: $(BIN_DIR)/$(BINARY_NAME) ## Run the built binary
	$(BIN_DIR)/$(BINARY_NAME) $(ARGS)

clean: ## Clean build artifacts
	rm -rf $(BIN_DIR) coverage.out coverage.html

setup-tests: ## Clone Ethereum test fixtures
	[ -d tests/fixtures ] || git clone --depth 1 https://github.com/ethereum/tests.git tests/fixtures
	[ -d tests/fixtures/GeneralStateTests ] || tar -xzf tests/fixtures/fixtures_general_state_tests.tgz -C tests/fixtures

test-unit: ## Run Go unit tests
	go test -race -count=1 ./internal/... ./cmd/...

test-e2e: ## Run E2E tests (requires setup-tests)
	go test -v ./tests/e2e/...

test: test-unit test-e2e ## Run all tests
