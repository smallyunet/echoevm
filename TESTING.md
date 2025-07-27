# EchoEVM Testing Guide

This directory contains comprehensive testing scripts for the EchoEVM project.

## Test Scripts

### 1. `test.sh` - Basic Test Script
The original test script with enhanced functionality. Includes organized test sections:

- Binary file execution tests
- Data type operations (arithmetic, boolean, factorial)
- Control flow tests (if-else, loops, require statements)
- Edge cases and boundary testing
- Performance testing
- Error handling scenarios

**Usage:**
```bash
./test.sh
```

### 2. `test_advanced.sh` - Advanced Test Script
Enhanced test script with better error handling, colored output, and detailed reporting:

Features:
- ✅ Pass/fail indicators with colors
- 📊 Test result summary
- 🚫 Expected failure testing
- 📝 Detailed output capture
- ⚡ Performance tracking

**Usage:**
```bash
./test_advanced.sh
```

### 3. `test_config.toml` - Test Configuration
Configuration file containing test cases and expected behaviors for systematic testing.

## Test Categories

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

### Edge Cases
- **Large Numbers**: Testing with big integers
- **Zero Values**: Boundary condition testing
- **Division Edge Cases**: Division by 1, large dividends
- **Performance Limits**: Testing computational boundaries

## Smart Contracts Tested

The test scripts cover contracts from multiple categories:

1. **01-data-types/**
   - `Add.sol` - Basic addition
   - `Sub.sol` - Basic subtraction
   - `IntegerTypes.sol` - Comprehensive integer operations
   - `BoolType.sol` - Boolean type operations
   - `Fact.sol` - Factorial calculations

2. **03-control-flow/**
   - `Require.sol` - Require statement testing
   - `IfElse.sol` - Conditional logic
   - `Loops.sol` - Loop constructs

## Running Tests

### Prerequisites
- Go environment set up
- EchoEVM project compiled
- All contract artifacts built (run `make` or build contracts)

### Quick Start
```bash
# Make scripts executable (if not already)
chmod +x test.sh test_advanced.sh

# Run basic tests
./test.sh

# Run advanced tests with detailed reporting
./test_advanced.sh
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

## Adding New Tests

To add new test cases:

1. **For basic testing**: Add new `go run` commands to `test.sh`
2. **For advanced testing**: Use the `run_test` or `run_failing_test` functions in `test_advanced.sh`
3. **For configuration**: Add entries to `test_config.toml`

### Example New Test
```bash
run_test "My new test description" \
    'go run ./cmd/echoevm run -artifact ./path/to/contract.json -function "myFunction(uint256)" -args "123"'
```

## Troubleshooting

### Common Issues
1. **Permission denied**: Run `chmod +x test.sh test_advanced.sh`
2. **Contract not found**: Ensure contracts are built with `cd test/contract && npm run compile`
3. **Go build errors**: Ensure Go environment is properly set up

### Debug Mode
For detailed debugging, you can run individual commands manually:
```bash
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -function "add(uint256,uint256)" -args "1,2"
```

## Contributing

When adding new contracts or test cases:
1. Update the appropriate test script
2. Add documentation for new test categories
3. Ensure tests have clear descriptions
4. Include both positive and negative test cases
5. Update this README if new test categories are added
