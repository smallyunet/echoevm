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

ECHOEVM_CMD="go run ./cmd/echoevm"

# Helper: deploy constructor bytecode (bin or artifact) and capture runtime hex
deploy_runtime() {
    local source_type=$1   # bin|artifact
    local path=$2
    local out
    if [ "$source_type" = "bin" ]; then
        if ! out=$($ECHOEVM_CMD deploy -b "$path" --print 2>/dev/null); then
            echo ""; return 1
        fi
    else
        if ! out=$($ECHOEVM_CMD deploy -a "$path" --print 2>/dev/null); then
            echo ""; return 1
        fi
    fi
    echo "$out" | tail -n 1
}

# Helper: call runtime code with function + args
call_runtime() {
    local runtime_hex=$1
    local function_sig=$2
    local args_str=$3
    # Write runtime to temp file to reuse existing call interface (expects file or artifact)
    local tmpfile
    tmpfile=$(mktemp)
    echo -n "$runtime_hex" > "$tmpfile"
    $ECHOEVM_CMD call -r "$tmpfile" -f "$function_sig" -A "$args_str"
    local status=$?
    rm -f "$tmpfile"
    return $status
}

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
    shift
    echo -e "\n${YELLOW}Testing: $name${NC}"
    if [ "$VERBOSE" = true ]; then
        echo -e "${BLUE}Command: $*${NC}"
        if "$@"; then
            echo -e "${GREEN}✓ PASSED${NC}"; ((PASSED++))
        else
            echo -e "${RED}✗ FAILED${NC}"; ((FAILED++))
        fi
    else
        if "$@" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ PASSED${NC}"; ((PASSED++))
        else
            echo -e "${RED}✗ FAILED${NC}"; ((FAILED++))
        fi
    fi
}

# Helper: execute function in binary (.bin) constructor then call runtime
bin_function_test() {
    local bin_file="$1"; shift
    local signature="$1"; shift
    local args="$1"; shift || true
    local runtime
    runtime=$(deploy_runtime bin "$bin_file") || return 1
    [ -n "$runtime" ] || return 1
    call_runtime "$runtime" "$signature" "$args"
}

# Helper: execute function using artifact (constructor->runtime->call)
artifact_function_test() {
    local artifact_file="$1"; shift
    local signature="$1"; shift
    local args="$1"; shift || true
    local runtime
    runtime=$(deploy_runtime artifact "$artifact_file") || return 1
    [ -n "$runtime" ] || return 1
    call_runtime "$runtime" "$signature" "$args"
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
    run_test "Addition" bin_function_test "$BINARY_DIR/build/Add_sol_Add.bin" 'add(uint256,uint256)' '1,2'
    run_test "Multiplication" bin_function_test "$BINARY_DIR/build/Multiply_sol_Multiply.bin" 'multiply(uint256,uint256)' '3,4'
    run_test "Summation" bin_function_test "$BINARY_DIR/build/Sum_sol_Sum.bin" 'sum(uint256)' '5'
}



# Main execution logic
case "$MODE" in
    "binary")
        run_binary_tests
        ;;
    "contract"|"all")
        echo -e "\n${BOLD}Ethereum Official Tests${NC}"
        echo "----------------------------------------"
        if go test -v ./tests/...; then
             echo -e "${GREEN}✓ PASSED${NC}"; ((PASSED++))
        else
             echo -e "${RED}✗ FAILED${NC}"; ((FAILED++))
        fi
        
        if [ "$MODE" = "all" ]; then
            run_binary_tests
        fi
        ;;
esac

# Output results
echo -e "\n${BOLD}${BLUE}=========================================${NC}"
echo -e "${BOLD}Test Results${NC}"
if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}${BOLD}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}${BOLD}❌ Some tests failed!${NC}"
    exit 1
fi
