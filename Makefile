BINARY_NAME ?= echoevm
BIN_DIR ?= bin

.PHONY: install build run test clean

install: ## Install the echoevm binary to GOPATH/bin
	go install ./cmd/echoevm

$(BIN_DIR)/$(BINARY_NAME): ## Build the echoevm binary
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/echoevm

build: $(BIN_DIR)/$(BINARY_NAME)

run: $(BIN_DIR)/$(BINARY_NAME) ## Run the built binary
	$(BIN_DIR)/$(BINARY_NAME) $(ARGS)

test: $(BIN_DIR)/$(BINARY_NAME) ## Run integration tests
	./test.sh

test-advanced: $(BIN_DIR)/$(BINARY_NAME) ## Run advanced tests with detailed reporting
	./test_advanced.sh

test-unit: ## Run Go unit tests
	go test ./...

test-all: test-unit test-advanced ## Run all tests (unit + integration)

clean: ## Clean build artifacts
	rm -rf $(BIN_DIR)
