#!/bin/bash

# EchoEVM Advanced Test Script
# This script provides comprehensive testing with better error handling and reporting

set -e  # Exit on any error

# Get the script directory to handle relative paths correctly
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a test and capture result
run_test() {
    local test_name="$1"
    local command="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo -e "${BLUE}Running:${NC} $test_name"
    echo -e "${YELLOW}Command:${NC} $command"
    
    if eval "$command" > /tmp/test_output.txt 2>&1; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        echo -e "${GREEN}✓ PASSED${NC}"
        echo -e "${YELLOW}Output:${NC}"
        cat /tmp/test_output.txt
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo -e "${RED}✗ FAILED${NC}"
        echo -e "${RED}Error output:${NC}"
        cat /tmp/test_output.txt
    fi
    echo "----------------------------------------"
}

# Function to run a test that is expected to fail
run_failing_test() {
    local test_name="$1"
    local command="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo -e "${BLUE}Running (Expected to fail):${NC} $test_name"
    echo -e "${YELLOW}Command:${NC} $command"
    
    if eval "$command" > /tmp/test_output.txt 2>&1; then
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo -e "${RED}✗ UNEXPECTED PASS (Should have failed)${NC}"
    else
        PASSED_TESTS=$((PASSED_TESTS + 1))
        echo -e "${GREEN}✓ FAILED AS EXPECTED${NC}"
    fi
    echo -e "${YELLOW}Output:${NC}"
    cat /tmp/test_output.txt
    echo "----------------------------------------"
}

echo "========================================="
echo -e "${BLUE}EchoEVM Advanced Test Suite${NC}"
echo "========================================="

# Check if echoevm binary exists
if [ ! -f "./cmd/echoevm/main.go" ]; then
    echo -e "${RED}Error: echoevm source not found${NC}"
    exit 1
fi

# 1. Binary file tests
echo -e "\n${YELLOW}1. Binary File Tests${NC}"
echo "========================================="

run_test "Basic binary execution" \
    'go run ./cmd/echoevm run -bin ./test/bins/build/Add_sol_Add.bin -function "add(uint256,uint256)" -args "1,2"'

# 2. Data Types Tests
echo -e "\n${YELLOW}2. Data Types Tests${NC}"
echo "========================================="

run_test "Basic addition" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -function "add(uint256,uint256)" -args "42,58"'

run_test "Basic subtraction" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Sub.sol/Sub.json -function "sub(uint256,uint256)" -args "100,30"'

run_test "Integer operations - divide" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/IntegerTypes.sol/IntegerTypes.json -function "divide(uint256,uint256)" -args "144,12"'

run_test "Integer operations - increment" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/IntegerTypes.sol/IntegerTypes.json -function "increment(uint256)" -args "999"'

run_test "Boolean operations - isActive" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/BoolType.sol/BoolType.json -function "isActive()"'

run_test "Factorial calculation (small)" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Fact.sol/Fact.json -function "fact(uint256)" -args "4"'

run_test "Factorial calculation (zero)" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Fact.sol/Fact.json -function "fact(uint256)" -args "0"'

# 3. Control Flow Tests
echo -e "\n${YELLOW}3. Control Flow Tests${NC}"
echo "========================================="

run_test "Require statement (valid input)" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Require.sol/Require.json -function "test(uint256)" -args "5"'

run_failing_test "Require statement (invalid input)" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Require.sol/Require.json -function "test(uint256)" -args "0"'

run_test "If-else logic (condition true)" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/IfElse.sol/IfElse.json -function "ifElse(uint256)" -args "5"'

run_test "If-else logic (condition false)" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/IfElse.sol/IfElse.json -function "ifElse(uint256)" -args "15"'

run_test "Conditional assignment (small value)" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/IfElse.sol/IfElse.json -function "conditionalAssignment(uint256)" -args "3"'

run_test "Complex conditional logic" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/IfElse.sol/IfElse.json -function "complexConditional(uint256)" -args "85"'

run_test "For loop execution" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Loops.sol/Loops.json -function "forLoop(uint256)" -args "7"'

run_test "Do-while loop execution" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Loops.sol/Loops.json -function "doWhileLoop(uint256)" -args "50"'

# 4. Edge Cases and Boundary Tests
echo -e "\n${YELLOW}4. Edge Cases and Boundary Tests${NC}"
echo "========================================="

run_test "Large number addition" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -function "add(uint256,uint256)" -args "999999999,1000000001"'

run_test "Zero addition" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -function "add(uint256,uint256)" -args "0,0"'

run_test "Division by one" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/IntegerTypes.sol/IntegerTypes.json -function "divide(uint256,uint256)" -args "12345,1"'

run_test "Large factorial" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Fact.sol/Fact.json -function "fact(uint256)" -args "8"'

# 5. Performance Tests
echo -e "\n${YELLOW}5. Performance Tests${NC}"
echo "========================================="

run_test "Medium loop performance" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Loops.sol/Loops.json -function "forLoop(uint256)" -args "25"'

run_test "Larger factorial performance" \
    'go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Fact.sol/Fact.json -function "fact(uint256)" -args "10"'

# Test Summary
echo -e "\n========================================="
echo -e "${BLUE}Test Summary${NC}"
echo "========================================="
echo -e "Total tests run: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed!${NC}"
    exit 1
fi
