 BINARY_NAME ?= echoevm
 BIN_DIR ?= bin

GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION    ?= dev
LDFLAGS    := -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE) -X main.Version=$(VERSION)

.PHONY: install build run test test-binary test-contract test-unit test-all coverage clean

install: ## Install the echoevm binary to GOPATH/bin
	go install -ldflags "$(LDFLAGS)" ./cmd/echoevm

$(BIN_DIR)/$(BINARY_NAME): ## Build the echoevm binary (with version ldflags)
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/echoevm

build: $(BIN_DIR)/$(BINARY_NAME)

run: $(BIN_DIR)/$(BINARY_NAME) ## Run the built binary
	$(BIN_DIR)/$(BINARY_NAME) $(ARGS)

## --------------------------------------------------
## Testing Targets
## --------------------------------------------------
# The original Makefile referenced test/scripts/* which do not exist.
# We standardize on a single orchestrator script: test/test.sh
# Provide granular targets for CI and local usage.

test: test-binary ## Default test target (binary quick tests)

test-binary: $(BIN_DIR)/$(BINARY_NAME) ## Run precompiled EVM binary contract tests (binary/*.bin)
	./test/test.sh --binary

test-contract: $(BIN_DIR)/$(BINARY_NAME) ## Run Hardhat artifact based contract tests
	./test/test.sh --contract

test-all: $(BIN_DIR)/$(BINARY_NAME) ## Run all integration (binary + contract + block) tests
	./test/test.sh --verbose
	./test/test_block.sh
	./test/test_block_run.sh

test-block: $(BIN_DIR)/$(BINARY_NAME) ## Run block apply integration tests
	./test/test_block.sh
	./test/test_block_run.sh

test-unit: ## Run Go unit tests
	go test -race -count=1 ./...

coverage: ## Run Go unit tests with coverage report (coverage.out + html)
	go test -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Coverage summary:" && go tool cover -func=coverage.out | tail -n 1
	@echo "Generate HTML report: go tool cover -html=coverage.out -o coverage.html"

clean: ## Clean build artifacts
	rm -rf $(BIN_DIR)
