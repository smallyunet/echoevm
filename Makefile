BINARY_NAME ?= echoevm
BIN_DIR ?= bin

GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION    ?= v0.4.1
LDFLAGS    := -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE) -X main.Version=$(VERSION)

.PHONY: install build build-chrome run test setup-tests setup-official-fixtures test-unit test-integration test-e2e test-chrome test-compliance test-official-fixtures test-regression test-conformance test-conformance-full test-deploy test-skills package-skills coverage clean help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

install: ## Install the echoevm binary to GOPATH/bin
	go install -ldflags "$(LDFLAGS)" ./cmd/echoevm

build: ## Build the echoevm binary
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/echoevm

build-chrome: ## Build the unpacked Chrome extension and release ZIP
	bash extensions/chrome/scripts/build.sh

run: build ## Run the built binary
	$(BIN_DIR)/$(BINARY_NAME) $(ARGS)

clean: ## Clean build artifacts
	rm -rf $(BIN_DIR) dist coverage.out coverage.html

setup-tests: ## Show bundled and full official fixture locations
	@echo "Curated fixtures: tests/compliance/fixtures"
	@echo "Full official EEST release: make setup-official-fixtures"

setup-official-fixtures: ## Download and verify the pinned official EEST release (~404 MiB)
	go run ./tests/official/cmd/fetch-fixtures

test-unit: ## Run Go unit tests
	go test -race -count=1 ./internal/... ./cmd/...

test-integration: ## Run integration tests
	go test -v ./tests/integration/...

test-e2e: ## Run CLI end-to-end tests
	go test -v ./tests/e2e/...

test-chrome: ## Validate Chrome UI helpers and the packaged Wasm engine
	node --test extensions/chrome/test/*.test.cjs
	GOOS=js GOARCH=wasm go test -count=1 -exec="bash $$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./internal/replay
	bash extensions/chrome/scripts/build.sh

test-compliance: ## Run compliance tests
	go test -v ./tests/compliance/...

test-official-fixtures: setup-official-fixtures ## Audit the release and execute Prague/Osaka fixtures under current Osaka rules
	ECHOEVM_OFFICIAL_FIXTURES=$(CURDIR)/tests/official/fixtures go test -v ./tests/official/...

test-regression: ## Run EchoEVM fork and execution regression vectors
	go test -v ./tests/differential/...

test-conformance: test-compliance test-regression ## Run curated conformance and independent execution regressions

test-conformance-full: test-conformance test-official-fixtures ## Run normal conformance plus the full official fixture audit

test-deploy: ## Validate the production deployment contract
	bash -n deploy/deploy-image.sh deploy/deploy-ssh-command.sh deploy/deploy-image_test.sh
	bash deploy/deploy-image_test.sh

test-skills: ## Validate portable Agent Skills and Claude mirrors
	python3 tools/validate_agent_skills.py
	python3 tools/sync_agent_skills.py --check
	python3 -m unittest discover -s tools -p 'test_*.py'

package-skills: test-skills ## Build portable .skill archives
	python3 tools/package_agent_skills.py

test: test-unit test-integration test-e2e test-chrome test-conformance test-deploy test-skills ## Run all tests
