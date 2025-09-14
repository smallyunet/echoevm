# EchoEVM Testing Guide

## Quick Start

EchoEVM now uses a simplified test script to run all tests with one command:

```bash
# Run all tests
./test/test.sh

# Run binary tests only (fast)
./test/test.sh --binary

# Run contract tests only (comprehensive)
./test/test.sh --contract

# Show detailed output
./test/test.sh --verbose

# Show help
./test/test.sh --help
```

## Test Content

### Binary Tests (3 tests)
- Addition
- Multiplication
- Summation

### Contract Tests (8 tests)
- Data Types: Addition, Subtraction, Factorial
- Control Flow: Require Pass, Require Fail, IfElse
- Functions: Visibility
- Events: Handling

## Test Structure

```
test/
├── test.sh              # Main test script (only one you need to use)
├── binary/              # Binary test files
│   ├── Add.sol
│   ├── Multiply.sol
│   ├── Sum.sol
│   └── build.sh
├── contract/            # Contract test files
│   ├── contracts/
│   ├── artifacts/
│   └── ...
└── README.md           # This file
```

## Output Example

```
=========================================
EchoEVM Test Suite
=========================================

Binary Tests
----------------------------------------

Testing: Addition
✓ PASSED

Testing: Multiplication
✓ PASSED

Testing: Summation
✓ PASSED

Contract Tests
----------------------------------------

Testing: Data Types - Addition
✓ PASSED

Testing: Data Types - Subtraction
✓ PASSED

Testing: Data Types - Factorial
✓ PASSED

Testing: Control Flow - Require Pass
✓ PASSED

Testing: Control Flow - Require Fail
✓ PASSED

Testing: Control Flow - IfElse
✓ PASSED

Testing: Function Visibility
✓ PASSED

Testing: Event Handling
✓ PASSED

=========================================
Test Results
Passed: 11
Failed: 0
Total: 11

🎉 All tests passed!
```

## Adding New Tests

To add new tests, simply edit the `test/test.sh` file:

1. Add binary tests in the `run_binary_tests()` function
2. Add contract tests in the `run_contract_tests()` function
3. Use the format: `run_test "Test Name" "command"`

It's that simple!
