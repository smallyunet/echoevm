BINARY_NAME ?= echoevm
BIN_DIR ?= bin

.PHONY: install build run run-rpc test test-binary test-contract test-unit test-all coverage clean

install: ## Install the echoevm binary to GOPATH/bin
	go install ./cmd/echoevm

$(BIN_DIR)/$(BINARY_NAME): ## Build the echoevm binary
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/echoevm

build: $(BIN_DIR)/$(BINARY_NAME)

run: $(BIN_DIR)/$(BINARY_NAME) ## Run the built binary
	$(BIN_DIR)/$(BINARY_NAME) $(ARGS)

run-rpc: $(BIN_DIR)/$(BINARY_NAME) ## Start the RPC server
	$(BIN_DIR)/$(BINARY_NAME) serve -http ${HTTP_ADDR:-localhost:8545}

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

test-all: $(BIN_DIR)/$(BINARY_NAME) ## Run all integration (binary + contract) tests
	./test/test.sh --verbose

test-unit: ## Run Go unit tests
	go test -race -count=1 ./...

coverage: ## Run Go unit tests with coverage report (coverage.out + html)
	go test -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Coverage summary:" && go tool cover -func=coverage.out | tail -n 1
	@echo "Generate HTML report: go tool cover -html=coverage.out -o coverage.html"

clean: ## Clean build artifacts
	rm -rf $(BIN_DIR)
