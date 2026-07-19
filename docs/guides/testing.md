# EchoEVM Testing Quick Start

## Quick Commands

```bash
# Run all tests
make test

# Go unit tests only (with race detector)
make test-unit

# Coverage report (Go packages)
make coverage

# Official fixtures plus geth differential vectors
make test-conformance
```

## Test Fixtures

EchoEVM keeps a small, pinned subset adapted from the official Ethereum
execution tests in `tests/compliance/fixtures/`. The source repository, commit,
file, fork, and behavior category are recorded in each fixture, so compliance
tests run offline and do not silently skip when an external checkout is missing.

The differential suite in `tests/differential/` executes the same Cancun
vectors with EchoEVM and go-ethereum, then compares return data, gas used, halt
class, and persistent storage. See
[EVM Conformance](conformance.md) for the baseline and CI contracts.

## Test Structure

```
tests/
├── compliance/         # Pinned Ethereum fixtures and runner
├── differential/       # Cancun behavior compared with go-ethereum
├── e2e/                # Built CLI behavior tests
└── integration/        # Cross-package behavior tests

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
