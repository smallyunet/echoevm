BINARY_NAME ?= echoevm
BIN_DIR ?= bin

GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION    ?= v0.0.24
LDFLAGS    := -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE) -X main.Version=$(VERSION)

.PHONY: install build run test test-unit test-integration test-e2e test-compliance test-differential test-conformance coverage clean help

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

setup-tests: ## Show compliance fixture location (fixtures are bundled)
	@echo "Compliance fixtures are bundled in tests/compliance/fixtures."

test-unit: ## Run Go unit tests
	go test -race -count=1 ./internal/... ./cmd/...

test-integration: ## Run integration tests
	go test -v ./tests/integration/...

test-e2e: ## Run CLI end-to-end tests
	go test -v ./tests/e2e/...

test-compliance: ## Run compliance tests
	go test -v ./tests/compliance/...

test-differential: ## Compare Cancun execution results with go-ethereum
	go test -v ./tests/differential/...

test-conformance: test-compliance test-differential ## Run official fixtures and geth differential tests

test: test-unit test-integration test-e2e test-conformance ## Run all tests
