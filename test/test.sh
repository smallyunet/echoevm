#!/bin/bash

# EchoEVM Test Script - Run all tests with one command
# Usage: ./test/test.sh [options]

set -uo pipefail

# Get project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

# Configuration
BINARY_DIR="$PROJECT_ROOT/test/binary"
CONTRACT_DIR="$PROJECT_ROOT/test/contract"
ECHOEVM_CMD="go run ./cmd/echoevm"

# Parse arguments
MODE="all"  # all, binary, contract
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --binary)
            MODE="binary"
            shift
            ;;
        --contract)
            MODE="contract"
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            echo "EchoEVM Test Script"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --binary      Run binary tests only"
            echo "  --contract    Run contract tests only"
            echo "  --verbose     Show detailed output"
            echo "  --help        Show this help message"
            echo ""
            echo "Default: Run all tests"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo -e "${BOLD}${BLUE}=========================================${NC}"
echo -e "${BOLD}${BLUE}EchoEVM Test Suite${NC}"
echo -e "${BOLD}${BLUE}=========================================${NC}"

# Test result tracking
PASSED=0
FAILED=0

# Function to run a test
run_test() {
    local name="$1"
    local cmd="$2"
    
    echo -e "\n${YELLOW}Testing: $name${NC}"
    
    if [ "$VERBOSE" = true ]; then
        echo -e "${BLUE}Command: $cmd${NC}"
    fi
    
    if eval "$cmd" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAILED${NC}"
        ((FAILED++))
    fi
}

# Binary tests
run_binary_tests() {
    echo -e "\n${BOLD}Binary Tests${NC}"
    echo "----------------------------------------"
    
    # Check and compile binary files
    if [ ! -d "$BINARY_DIR/build" ]; then
        echo "Compiling binary contracts..."
        (cd "$BINARY_DIR" && ./build.sh)
    fi
    
    # Basic arithmetic tests
    run_test "Addition" "$ECHOEVM_CMD run -bin $BINARY_DIR/build/Add_sol_Add.bin -function 'add(uint256,uint256)' -args '1,2'"
    run_test "Multiplication" "$ECHOEVM_CMD run -bin $BINARY_DIR/build/Multiply_sol_Multiply.bin -function 'multiply(uint256,uint256)' -args '3,4'"
    run_test "Summation" "$ECHOEVM_CMD run -bin $BINARY_DIR/build/Sum_sol_Sum.bin -function 'sum(uint256)' -args '5'"
}

# Contract tests
run_contract_tests() {
    echo -e "\n${BOLD}Contract Tests${NC}"
    echo "----------------------------------------"
    
    # Check and compile contracts
    if [ ! -d "$CONTRACT_DIR/artifacts" ]; then
        echo "Compiling contracts..."
        (cd "$CONTRACT_DIR" && npm run compile)
    fi
    
    local artifacts="$CONTRACT_DIR/artifacts/contracts"
    
    # Data type tests
    run_test "Data Types - Addition" "$ECHOEVM_CMD run -artifact $artifacts/01-data-types/Add.sol/Add.json -function 'add(uint256,uint256)' -args '10,20'"
    run_test "Data Types - Subtraction" "$ECHOEVM_CMD run -artifact $artifacts/01-data-types/Sub.sol/Sub.json -function 'sub(uint256,uint256)' -args '50,20'"
    run_test "Data Types - Factorial" "$ECHOEVM_CMD run -artifact $artifacts/01-data-types/Fact.sol/Fact.json -function 'fact(uint256)' -args '5'"
    
    # Control flow tests
    run_test "Control Flow - Require Pass" "$ECHOEVM_CMD run -artifact $artifacts/03-control-flow/Require.sol/Require.json -function 'test(uint256)' -args '5'"
    # This test expects failure, so invert logic
    echo -e "\n${YELLOW}Testing: Control Flow - Require Fail${NC}"
    if [ "$VERBOSE" = true ]; then
        echo -e "${BLUE}Command: $ECHOEVM_CMD run -artifact $artifacts/03-control-flow/Require.sol/Require.json -function 'test(uint256)' -args '0'${NC}"
    fi
    if ! $ECHOEVM_CMD run -artifact "$artifacts/03-control-flow/Require.sol/Require.json" -function "test(uint256)" -args "0" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAILED${NC}"
        ((FAILED++))
    fi
    run_test "Control Flow - IfElse" "$ECHOEVM_CMD run -artifact $artifacts/03-control-flow/IfElse.sol/IfElse.json -function 'ifElse(uint256)' -args '5'"
    
    # Function tests
    run_test "Function Visibility" "$ECHOEVM_CMD run -artifact $artifacts/02-functions/FunctionVisibility.sol/FunctionVisibility.json -function 'publicFunction()'"
    
    # Event tests
    run_test "Event Handling" "$ECHOEVM_CMD run -artifact $artifacts/05-events/Lock.sol/Lock.json -function 'withdraw()'"
}

# Main execution logic
case "$MODE" in
    "binary")
        run_binary_tests
        ;;
    "contract")
        run_contract_tests
        ;;
    "all")
        run_binary_tests
        run_contract_tests
        ;;
esac

# Output results
echo -e "\n${BOLD}${BLUE}=========================================${NC}"
echo -e "${BOLD}Test Results${NC}"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo -e "${BOLD}Total: $((PASSED + FAILED))${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}${BOLD}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}${BOLD}❌ Some tests failed!${NC}"
    exit 1
fi
