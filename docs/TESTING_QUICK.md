# EchoEVM Testing Quick Start

## Quick Commands

```bash
# Run all tests
make test

# Go unit tests only (with race detector)
make test-unit

# Coverage report (Go packages)
make coverage
```

## Test Fixtures

EchoEVM uses the [ethereum/tests](https://github.com/ethereum/tests) repository as a git submodule for test fixtures.

```bash
# Initialize test fixtures (required before running some tests)
make setup-tests
```

This will clone the Ethereum test fixtures into `tests/fixtures/`.

## Test Structure

```
tests/
├── fixtures/           # Git submodule: ethereum/tests
├── state_test.go       # State transition tests
└── genesis_test.json   # Genesis configuration for tests

internal/
├── evm/core/           # Core tests (stack, memory, genesis, opcodes)
└── evm/vm/             # VM tests (interpreter, operations)
```

## Running Specific Tests

```bash
# Run a specific test file
go test -v ./internal/evm/core/stack_test.go

# Run tests matching a pattern
go test -v ./... -run TestOpAdd

# Run with race detector
go test -race ./...
```

## Writing Tests

When adding new features:
1. Add unit tests in the relevant `*_test.go` file
2. Run `make test` to verify all tests pass
3. Run `make coverage` to check test coverage
