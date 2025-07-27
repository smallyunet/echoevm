# EchoEVM Testing

For comprehensive testing documentation, see [`tests/docs/README.md`](./tests/docs/README.md).

## Quick Start

```bash
# Run all tests
make test-run-all

# Run specific test suites
make test-unit      # Unit tests only
make test           # Basic integration tests
make test-advanced  # Advanced integration tests with detailed reporting
```

## Test Directory Structure

```
tests/
├── scripts/           # Test execution scripts
├── config/           # Test configuration files
├── docs/             # Detailed documentation
└── reports/          # Auto-generated test reports
```

## Legacy Test Files

The following files have been reorganized:
- `test.sh` → `tests/scripts/basic.sh`
- `test_advanced.sh` → `tests/scripts/advanced.sh`  
- `test_config.toml` → `tests/config/test_cases.toml`
- `TESTING.md` → `tests/docs/README.md`

For detailed information, see the [comprehensive testing guide](./tests/docs/README.md).
