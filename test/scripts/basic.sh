#!/bin/bash

# EchoEVM Basic Test Script
# This script tests various smart contract functionalities using the EchoEVM

# Get the script directory to handle relative paths correctly
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

echo "========================================="
echo "EchoEVM Comprehensive Test Suite"
echo "========================================="

# Test with binary file (original test)
echo ""
echo "1. Testing with binary file:"
echo "---------------------------------------------"
go run ./cmd/echoevm run -bin ./test/bins/build/Add_sol_Add.bin -function "add(uint256,uint256)" -args "1,2"

# 01-data-types: Basic data type operations
echo ""
echo "2. Testing Data Types:"
echo "---------------------------------------------"
echo "2.1 Basic arithmetic operations:"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -function "add(uint256,uint256)" -args "1,2"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Sub.sol/Sub.json -function "sub(uint256,uint256)" -args "5,3"

echo "2.2 Integer types operations:"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/IntegerTypes.sol/IntegerTypes.json -function "add(uint256,uint256)" -args "10,20"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/IntegerTypes.sol/IntegerTypes.json -function "divide(uint256,uint256)" -args "100,4"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/IntegerTypes.sol/IntegerTypes.json -function "increment(uint256)" -args "5"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/IntegerTypes.sol/IntegerTypes.json -function "decrement(uint256)" -args "10"

echo "2.3 Boolean operations:"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/BoolType.sol/BoolType.json -function "isActive()" -args ""
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/BoolType.sol/BoolType.json -function "getActiveStatus()" -args ""

echo "2.4 Factorial calculation:"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Fact.sol/Fact.json -function "fact(uint256)" -args "5"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Fact.sol/Fact.json -function "fact(uint256)" -args "0"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Fact.sol/Fact.json -function "fact(uint256)" -args "6"

# 03-control-flow: Control flow operations
echo ""
echo "3. Testing Control Flow:"
echo "---------------------------------------------"
echo "3.1 Require statements (should pass):"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Require.sol/Require.json -function "test(uint256)" -args "1"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Require.sol/Require.json -function "test(uint256)" -args "100"

echo "3.2 Require statements (should fail):"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Require.sol/Require.json -function "test(uint256)" -args "0"

echo "3.3 If-Else conditions:"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/IfElse.sol/IfElse.json -function "ifElse(uint256)" -args "5"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/IfElse.sol/IfElse.json -function "ifElse(uint256)" -args "15"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/IfElse.sol/IfElse.json -function "conditionalAssignment(uint256)" -args "3"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/IfElse.sol/IfElse.json -function "conditionalAssignment(uint256)" -args "8"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/IfElse.sol/IfElse.json -function "complexConditional(uint256)" -args "25"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/IfElse.sol/IfElse.json -function "complexConditional(uint256)" -args "75"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/IfElse.sol/IfElse.json -function "complexConditional(uint256)" -args "95"

echo "3.4 Loop operations:"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Loops.sol/Loops.json -function "forLoop(uint256)" -args "5"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Loops.sol/Loops.json -function "forLoop(uint256)" -args "10"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Loops.sol/Loops.json -function "doWhileLoop(uint256)" -args "100"

# Additional edge cases and boundary testing
echo ""
echo "4. Testing Edge Cases:"
echo "---------------------------------------------"
echo "4.1 Large numbers:"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -function "add(uint256,uint256)" -args "1000000,2000000"

echo "4.2 Zero values:"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -function "add(uint256,uint256)" -args "0,0"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Sub.sol/Sub.json -function "sub(uint256,uint256)" -args "100,0"

echo "4.3 Division edge cases:"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/IntegerTypes.sol/IntegerTypes.json -function "divide(uint256,uint256)" -args "1,1"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/IntegerTypes.sol/IntegerTypes.json -function "divide(uint256,uint256)" -args "100,1"

# Performance and stress testing
echo ""
echo "5. Performance Testing:"
echo "---------------------------------------------"
echo "5.1 Loop performance:"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Loops.sol/Loops.json -function "forLoop(uint256)" -args "50"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Loops.sol/Loops.json -function "forLoop(uint256)" -args "100"

echo "5.2 Factorial performance:"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Fact.sol/Fact.json -function "fact(uint256)" -args "8"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/01-data-types/Fact.sol/Fact.json -function "fact(uint256)" -args "10"

# Error handling and recovery testing
echo ""
echo "6. Error Handling Testing:"
echo "---------------------------------------------"
echo "6.1 Various require failure scenarios:"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/03-control-flow/Require.sol/Require.json -function "test(uint256)" -args "0" || echo "Expected failure - test passed"

echo ""
echo "========================================="
echo "Test Suite Completed"
echo "========================================="
