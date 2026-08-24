BINARY_NAME ?= echoevm
BIN_DIR ?= bin
VERSION ?= v1.6.0

.PHONY: help install build build-chrome run clean setup-official-fixtures test-unit test-bytecode-conformance test-integration test-e2e test-explain test-test-witness test-chrome test-official-fixtures test-conformance test-conformance-full test-deploy test-skills package-skills test

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

install: ## Install the Rust CLI with Cargo
	cargo install --path crates/echoevm-cli --locked

build: ## Build the release CLI
	@mkdir -p $(BIN_DIR)
	cargo build --release --locked -p echoevm
	cp target/release/$(BINARY_NAME) $(BIN_DIR)/$(BINARY_NAME)

build-chrome: ## Build the unpacked Chrome extension and release ZIP
	bash extensions/chrome/scripts/build.sh

run: build ## Run the built binary
	$(BIN_DIR)/$(BINARY_NAME) $(ARGS)

clean: ## Clean generated build artifacts
	cargo clean
	rm -rf $(BIN_DIR) dist build coverage.out coverage.html

setup-official-fixtures: ## Download and verify pinned EEST fixtures (~404 MiB)
	bash scripts/fetch-official-fixtures.sh

test-unit: ## Run Rust unit and integration tests
	cargo test --workspace --locked

test-bytecode-conformance: ## Run exact multi-fork bytecode regression vectors
	bash scripts/test-bytecode-conformance.sh

test-integration: ## Exercise CLI and Solidity editor protocol
	cargo build --locked -p echoevm
	ECHOEVM_TEST_BINARY=$(CURDIR)/target/debug/echoevm npm --prefix editors/vscode run test:integration

test-e2e: ## Exercise the public CLI surface
	bash scripts/test-cli.sh

test-explain: ## Regenerate and compare deterministic explain fixtures
	cargo build --locked -p echoevm
	bash scripts/test-explain.sh

test-test-witness: ## Exercise supported and rejected self-contained test witnesses
	cargo build --locked -p echoevm
	bash scripts/test-test-witness.sh

test-chrome: ## Validate and smoke-test the packaged Rust Wasm extension
	node --test extensions/chrome/test/*.test.cjs
	bash extensions/chrome/scripts/build.sh

test-official-fixtures: setup-official-fixtures ## Execute pinned multi-fork fixtures with zero skip
	bash scripts/test-official-fixtures.sh

test-conformance: test-unit test-bytecode-conformance test-e2e test-explain test-test-witness ## Run focused native conformance gates

test-conformance-full: test-conformance test-official-fixtures ## Add the full official corpus

test-deploy: ## Validate the production deployment contract
	bash -n deploy/deploy-image.sh deploy/deploy-ssh-command.sh deploy/deploy-image_test.sh
	bash deploy/deploy-image_test.sh

test-skills: ## Validate portable Agent Skills and mirrors
	python3 tools/validate_agent_skills.py
	python3 tools/sync_agent_skills.py --check
	python3 -m unittest discover -s tools -p 'test_*.py'

package-skills: test-skills ## Build portable .skill archives
	python3 tools/package_agent_skills.py

test: test-unit test-integration test-e2e test-explain test-test-witness test-chrome test-deploy test-skills ## Run normal release gates
