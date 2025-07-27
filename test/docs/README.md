# EchoEVM Testing Guide

This directory contains comprehensive testing scripts for the EchoEVM project.

## 📁 Test Directory Structure

```
tests/
├── scripts/           # Test execution scripts
│   ├── basic.sh      # Basic integration tests
│   ├── advanced.sh   # Advanced tests with detailed reporting
│   └── run_all.sh    # Master test runner
├── config/           # Test configuration files
│   ├── test_cases.toml    # Test case definitions
│   └── environments.toml # Environment configurations
├── docs/             # Testing documentation
│   └── README.md     # This file
└── reports/          # Test result reports (auto-generated)
    ├── latest.html
    └── history/
```

## 🚀 Quick Start

### Run All Tests
```bash
# From project root
make test-all

# Or directly
./tests/scripts/advanced.sh
```

### Run Specific Test Categories
```bash
# Basic tests only
./tests/scripts/basic.sh

# Advanced tests with detailed reporting
./tests/scripts/advanced.sh
```

## 📋 Test Scripts

### 1. `tests/scripts/basic.sh` - Basic Test Script
Comprehensive integration tests covering all major functionality:

- Binary file execution tests
- Data type operations (arithmetic, boolean, factorial)
- Control flow tests (if-else, loops, require statements)
- Edge cases and boundary testing
- Performance testing
- Error handling scenarios

**Features:**
- Simple pass/fail output
- Sequential test execution
- Basic error handling

### 2. `tests/scripts/advanced.sh` - Advanced Test Script
Enhanced test script with professional reporting and error handling:

**Features:**
- ✅ **Colored Output**: Pass/fail indicators with colors
- 📊 **Test Summary**: Detailed result statistics
- 🚫 **Expected Failures**: Proper handling of tests that should fail
- 📝 **Output Capture**: Detailed logging of all test outputs
- ⚡ **Performance Tracking**: Execution time monitoring
- 🔄 **Retry Logic**: Automatic retry for flaky tests

**Usage:**
```bash
cd /path/to/echoevm
./tests/scripts/advanced.sh
```

### 3. `tests/config/test_cases.toml` - Test Configuration
Structured configuration file containing test cases and expected behaviors:

```toml
[data_types.basic]
add_simple = {
    artifact = "./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json",
    function = "add(uint256,uint256)",
    args = "10,20",
    expected_success = true,
    description = "Simple addition test"
}
```

## 🧪 Test Categories

### Data Types Tests
- **Basic Arithmetic**: Addition, subtraction, division
- **Integer Operations**: Increment, decrement, various integer types
- **Boolean Operations**: Boolean state checks
- **Factorial Calculations**: Recursive algorithm testing

### Control Flow Tests
- **Require Statements**: Success and failure scenarios
- **If-Else Logic**: Various conditional branches
- **Loops**: For loops and do-while loops
- **Complex Conditionals**: Multi-branch logic

### Cryptographic Operations
- **SHA3/Keccak256**: Hash function testing with various data sizes
- **Memory Operations**: Reading and writing to memory for hash calculations

### Edge Cases
- **Large Numbers**: Testing with big integers
- **Zero Values**: Boundary condition testing
- **Division Edge Cases**: Division by 1, large dividends
- **Performance Limits**: Testing computational boundaries

## 🔧 Recent Fixes

### SHA3 (0x20) Opcode Implementation
**Fixed Issue**: `panic: unsupported opcode 0x20`

Previously, the SHA3 (Keccak256) opcode was defined but not implemented, causing crashes when smart contracts used hash functions (common in loops and complex logic).

**Implementation Details**:
- Added `opSha3` function in `op_sha3.go`
- Reads data from memory using offset and size from stack
- Uses `golang.org/x/crypto/sha3.NewLegacyKeccak256()` for Ethereum compatibility
- Pushes 32-byte hash result back to stack
- Includes comprehensive unit tests

**Affected Contracts**: Loops, complex conditionals, and any contract using hash functions.

## 🎯 Smart Contracts Tested

The test scripts cover contracts from multiple categories:

### 1. **01-data-types/**
- `Add.sol` - Basic addition
- `Sub.sol` - Basic subtraction
- `IntegerTypes.sol` - Comprehensive integer operations
- `BoolType.sol` - Boolean type operations
- `Fact.sol` - Factorial calculations

### 2. **03-control-flow/**
- `Require.sol` - Require statement testing
- `IfElse.sol` - Conditional logic
- `Loops.sol` - Loop constructs

## 🏃‍♂️ Running Tests

### Prerequisites
- Go environment set up
- EchoEVM project compiled
- All contract artifacts built

### Environment Setup
```bash
# Build contracts (if needed)
cd test/contract
npm install
npm run compile

# Return to project root
cd ../..

# Build EchoEVM
make build
```

### Test Execution
```bash
# Quick test run
make test

# Advanced test run with detailed reporting
make test-advanced

# Run only unit tests
make test-unit

# Run all tests (unit + integration)
make test-all
```

### Test Output
The advanced test script provides colored output:
- 🟢 **Green**: Passed tests
- 🔴 **Red**: Failed tests  
- 🟡 **Yellow**: Test descriptions and commands
- 🔵 **Blue**: Section headers

### Expected Results
- Most tests should **PASS** ✅
- Some tests are **expected to fail** (like require statements with invalid input) 🚫
- The summary shows total tests, passed, and failed counts

## ➕ Adding New Tests

### For Basic Testing
Add new test commands to `tests/scripts/basic.sh`:
```bash
echo "Testing new feature:"
go run ./cmd/echoevm run -artifact ./path/to/contract.json -function "newFunction(uint256)" -args "123"
```

### For Advanced Testing
Use the helper functions in `tests/scripts/advanced.sh`:
```bash
run_test "My new test description" \
    'go run ./cmd/echoevm run -artifact ./path/to/contract.json -function "myFunction(uint256)" -args "123"'

# For tests that should fail
run_failing_test "Test that should fail" \
    'go run ./cmd/echoevm run -artifact ./path/to/contract.json -function "badFunction(uint256)" -args "0"'
```

### For Configuration-Based Testing
Add entries to `tests/config/test_cases.toml`:
```toml
[new_category.new_test]
my_test = {
    artifact = "./test/contract/artifacts/contracts/MyContract.sol/MyContract.json",
    function = "myFunction(uint256)",
    args = "42",
    expected_success = true,
    description = "Description of what this test does"
}
```

## 🐛 Troubleshooting

### Common Issues
1. **Permission denied**: Run `chmod +x tests/scripts/*.sh`
2. **Contract not found**: Ensure contracts are built with `cd test/contract && npm run compile`
3. **Go build errors**: Ensure Go environment is properly set up
4. **Path issues**: Make sure you're running tests from the project root directory

### Debug Mode
For detailed debugging, you can run individual commands manually:
```bash
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -function "add(uint256,uint256)" -args "1,2"
```

### Verbose Output
Enable verbose logging:
```bash
ECHOEVM_LOG_LEVEL=debug ./tests/scripts/advanced.sh
```

## 🤝 Contributing

When adding new contracts or test cases:

1. **Update Tests**: Add test cases to appropriate script
2. **Document Changes**: Update this README if new categories are added
3. **Test Descriptions**: Ensure tests have clear, descriptive names
4. **Positive & Negative**: Include both success and failure test cases
5. **Configuration**: Add structured test definitions to TOML config
6. **Performance**: Consider performance implications of new tests

### Test Naming Conventions
- Use descriptive names: `"Factorial calculation (zero input)"` vs `"Test 1"`
- Include expected behavior: `"Require statement (should fail)"`
- Categorize properly: Data types, control flow, edge cases, etc.

### Review Checklist
- [ ] Tests pass locally
- [ ] Both positive and negative cases covered
- [ ] Documentation updated
- [ ] Configuration files updated if applicable
- [ ] Performance impact considered
