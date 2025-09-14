# EchoEVM Testing Quick Start

## Quick Commands

```bash
# Go unit tests (race detector)
make test-unit

# Binary (.bin) contract tests (fast smoke)
make test-binary   # or just: make test

# Hardhat artifact contract tests
make test-contract

# All integration tests (binary + contract, verbose)
make test-all

# Coverage (Go packages)
make coverage
```

## Manual Testing

```bash
# Unified test runner (all integration tests)
./test/test.sh

# Only binary contract tests
./test/test.sh --binary

# Only Hardhat artifact contract tests
./test/test.sh --contract

# Verbose mode
./test/test.sh --verbose
```

## Test Structure

Current structure (simplified):

```
test/
├── test.sh             # Unified runner
├── scripts/            # Backwards-compatible wrappers (basic/advanced/run_all)
├── binary/             # Simple sample .sol + build.sh -> .bin outputs
└── contract/           # Hardhat project (artifacts/, contracts/, tests/ etc.)
```

## Documentation

- **Quick Start**: This file
- **Comprehensive Guide**: [test/docs/README.md](../test/docs/README.md)
- **Contract Testing**: [test/contract/README.md](../test/contract/README.md)
- **Test Directory**: [test/README.md](../test/README.md)

## Legacy Files

Legacy mapping / compatibility:
- Older docs referenced `test/scripts/basic.sh` and `advanced.sh`; wrappers now delegate to `test/test.sh`.
	Prefer calling `test/test.sh` directly going forward.
