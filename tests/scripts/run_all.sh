#!/bin/bash

# EchoEVM Master Test Runner
# This script runs all available test suites

set -e

# Get the script directory to handle relative paths correctly
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Test suite options
RUN_UNIT_TESTS=true
RUN_BASIC_TESTS=true
RUN_ADVANCED_TESTS=true
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --unit-only)
            RUN_UNIT_TESTS=true
            RUN_BASIC_TESTS=false
            RUN_ADVANCED_TESTS=false
            shift
            ;;
        --integration-only)
            RUN_UNIT_TESTS=false
            RUN_BASIC_TESTS=false
            RUN_ADVANCED_TESTS=true
            shift
            ;;
        --basic-only)
            RUN_UNIT_TESTS=false
            RUN_BASIC_TESTS=true
            RUN_ADVANCED_TESTS=false
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            echo "EchoEVM Test Runner"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --unit-only        Run only unit tests"
            echo "  --integration-only Run only integration tests (advanced)"
            echo "  --basic-only       Run only basic integration tests"
            echo "  --verbose, -v      Enable verbose output"
            echo "  --help, -h         Show this help message"
            echo ""
            echo "Default: Run all test suites"
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
echo -e "${BOLD}${BLUE}EchoEVM Comprehensive Test Suite${NC}"
echo -e "${BOLD}${BLUE}=========================================${NC}"

# Track overall results
OVERALL_SUCCESS=true
TEST_RESULTS=""

# Function to run a test suite and track results
run_test_suite() {
    local suite_name="$1"
    local command="$2"
    local required="$3"  # true/false
    
    echo -e "\n${YELLOW}${BOLD}Running $suite_name...${NC}"
    echo "----------------------------------------"
    
    if [ "$VERBOSE" = true ]; then
        echo -e "${BLUE}Command: $command${NC}"
    fi
    
    if eval "$command"; then
        echo -e "${GREEN}✓ $suite_name PASSED${NC}"
        TEST_RESULTS="${TEST_RESULTS}✅ $suite_name: PASSED\n"
    else
        echo -e "${RED}✗ $suite_name FAILED${NC}"
        TEST_RESULTS="${TEST_RESULTS}❌ $suite_name: FAILED\n"
        if [ "$required" = true ]; then
            OVERALL_SUCCESS=false
        fi
    fi
}

# Check prerequisites
echo -e "${BLUE}Checking prerequisites...${NC}"

if [ ! -f "./cmd/echoevm/main.go" ]; then
    echo -e "${RED}Error: EchoEVM source not found${NC}"
    exit 1
fi

if [ ! -d "./test/contract/artifacts" ]; then
    echo -e "${YELLOW}Warning: Contract artifacts not found. Building contracts...${NC}"
    cd test/contract
    if command -v npm &> /dev/null; then
        npm install && npm run compile
    else
        echo -e "${RED}Error: npm not found. Please build contracts manually.${NC}"
        exit 1
    fi
    cd "$PROJECT_ROOT"
fi

echo -e "${GREEN}✓ Prerequisites satisfied${NC}"

# Start timer
START_TIME=$(date +%s)

# Run test suites based on options
if [ "$RUN_UNIT_TESTS" = true ]; then
    run_test_suite "Unit Tests" "go test ./... -v" true
fi

if [ "$RUN_BASIC_TESTS" = true ]; then
    # Make sure script is executable
    chmod +x tests/scripts/basic.sh
    run_test_suite "Basic Integration Tests" "./tests/scripts/basic.sh" false
fi

if [ "$RUN_ADVANCED_TESTS" = true ]; then
    # Make sure script is executable
    chmod +x tests/scripts/advanced.sh
    run_test_suite "Advanced Integration Tests" "./tests/scripts/advanced.sh" true
fi

# Calculate total time
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
MINUTES=$((DURATION / 60))
SECONDS=$((DURATION % 60))

# Final report
echo -e "\n${BOLD}${BLUE}=========================================${NC}"
echo -e "${BOLD}${BLUE}Test Suite Summary${NC}"
echo -e "${BOLD}${BLUE}=========================================${NC}"

echo -e "${TEST_RESULTS}"

echo -e "${BLUE}Total execution time: ${MINUTES}m ${SECONDS}s${NC}"

if [ "$OVERALL_SUCCESS" = true ]; then
    echo -e "\n${GREEN}${BOLD}🎉 ALL CRITICAL TESTS PASSED!${NC}"
    echo -e "${GREEN}EchoEVM is ready for use.${NC}"
    exit 0
else
    echo -e "\n${RED}${BOLD}❌ SOME CRITICAL TESTS FAILED!${NC}"
    echo -e "${RED}Please review the failures above before using EchoEVM.${NC}"
    exit 1
fi
