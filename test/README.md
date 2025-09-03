# EchoEVM Test Directory

This directory contains all test-related files for the EchoEVM project.

## Quick Start

For testing commands and quick start guide, see [docs/TESTING_QUICK.md](../docs/TESTING_QUICK.md).

## Directory Structure

```
test/
├── bins/                   # Binary contract files
├── contract/               # Hardhat contract development environment
├── scripts/                # Test execution scripts
├── config/                 # Test configuration files
├── utils/                  # Test utility functions
├── docs/                   # Test documentation
└── reports/                # Test reports
```

## Documentation

- **Quick Start**: [docs/TESTING_QUICK.md](../docs/TESTING_QUICK.md)
- **Comprehensive Guide**: [docs/README.md](docs/README.md)
- **Contract Testing**: [contract/README.md](contract/README.md)

## Development Tools

- **Contract Development**: Use the Hardhat environment in `contract/`
- **Binary Contracts**: Use pre-compiled contracts in `bins/`
- **Testing Tools**: Use helper functions in `utils/`

## Prerequisites

- Go environment set up
- Node.js for contract compilation
- EchoEVM project compiled (`make build`)
- Test scripts need execution permissions (`chmod +x test/scripts/*.sh`)
