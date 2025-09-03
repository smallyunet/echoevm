# EchoEVM Testing Quick Start

## Quick Commands

```bash
# Run all tests
make test-all

# Run specific test suites
make test-unit      # Unit tests only
make test           # Basic integration tests
make test-advanced  # Advanced integration tests with detailed reporting
```

## Manual Testing

```bash
# Test arithmetic operations
./test/scripts/basic.sh

# Advanced test scenarios
./test/scripts/advanced.sh
```

## Test Structure

```
test/
├── scripts/           # Test execution scripts
├── config/           # Test configuration files
├── docs/             # Detailed documentation
└── reports/          # Auto-generated test reports
```

## Documentation

- **Quick Start**: This file
- **Comprehensive Guide**: [test/docs/README.md](../test/docs/README.md)
- **Contract Testing**: [test/contract/README.md](../test/contract/README.md)
- **Test Directory**: [test/README.md](../test/README.md)

## Legacy Files

The following files have been reorganized:
- `test.sh` → `test/scripts/basic.sh`
- `test_advanced.sh` → `test/scripts/advanced.sh`  
- `test_config.toml` → `test/config/test_cases.toml`
- `TESTING.md` → `test/docs/README.md`
