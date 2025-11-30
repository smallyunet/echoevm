# Ethereum Official Tests Integration

This directory contains the integration of the official [Ethereum Tests](https://github.com/ethereum/tests) suite for EchoEVM.

## Setup

The tests are located in `fixtures/`, which is a git submodule. Initialize it using:

```bash
make setup-tests
```

Or manually:

```bash
git submodule update --init --depth 1 tests/fixtures
```

## Running Tests

Run the tests using the standard Go test command:

```bash
go test -v ./tests/...
```

Or use the Makefile:

```bash
make test
```

## Test Runner

The test runner is implemented in `state_test.go`. It runs the `GeneralStateTests` found in `fixtures/GeneralStateTests/stExample`.

### Limitations

*   **Gas Metering**: EchoEVM currently does not implement full gas metering. Therefore, balance checks in the tests are relaxed (logged as warnings instead of errors) if they mismatch due to gas costs.
*   **Opcodes**: Some recent opcodes (e.g., `MCOPY`, `TLOAD`, `TSTORE` from Cancun) or complex opcodes might not be fully implemented, causing some tests to revert.
*   **Forks**: The runner attempts to use "Cancun", "Shanghai", or other recent forks.

## Adding More Tests

To enable more tests, modify `tests/state_test.go` to point `fixturesDir` to other directories within `fixtures/GeneralStateTests`.
