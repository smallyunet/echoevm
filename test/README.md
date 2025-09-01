# EchoEVM Test Directory

This directory contains all test-related files for the EchoEVM project, including smart contracts, test scripts, and tools.

## Directory Structure

```
test/
├── bins/                   # Binary contract files
│   ├── Add.sol
│   ├── Multiply.sol
│   ├── Sum.sol
│   └── build/              # Compiled binary files
├── contract/               # Hardhat contract development environment
│   ├── contracts/          # Solidity source code
│   ├── artifacts/          # Compilation artifacts
│   ├── test/               # Hardhat unit tests
│   └── hardhat.config.ts   # Hardhat configuration
├── scripts/                # Test execution scripts
│   ├── basic.sh            # Basic tests
│   ├── advanced.sh         # Advanced tests
│   └── run_all.sh          # Run all tests
├── config/                 # Test configuration files
│   ├── test_cases.toml     # Test case definitions
│   └── environments.toml   # Environment configuration
├── utils/                  # Test utility functions
│   ├── helpers.sh          # Common helper functions
│   └── contract_utils.sh   # Contract-related tools
├── docs/                   # Test documentation
│   ├── TESTING_GUIDE.md    # Testing guide
│   └── examples/           # Test examples
└── reports/                # Test reports
    ├── latest/             # Latest test results
    └── history/            # Historical test records
```

## Usage

### Running Tests
```bash
# Run all tests
cd test/scripts && ./run_all.sh

# Run basic tests
cd test/scripts && ./basic.sh

# Run advanced tests
cd test/scripts && ./advanced.sh
```

### Adding New Tests
1. Define new test cases in `config/test_cases.toml`
2. Add new contracts in `contracts/` if needed
3. Run tests to verify

### Viewing Test Results
Test results are saved in the `reports/` directory, including:
- Execution logs
- Performance data
- Error reports

## Development Tools

- **Contract Development**: Use the Hardhat environment in the `contract/` directory
- **Binary Contracts**: Use pre-compiled contracts in the `bins/` directory
- **Testing Tools**: Use helper functions in the `utils/` directory

## Notes

1. Ensure necessary dependencies are installed (Go, Node.js, jq)
2. Compile the project before running tests: `make build`
3. Test scripts need execution permissions: `chmod +x test/scripts/*.sh`
