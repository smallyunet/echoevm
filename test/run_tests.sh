#!/bin/bash
set -euo pipefail

BUILD_DIR="$(dirname "$0")/../build"
CONTRACT_DIR="$(dirname "$0")/contracts"
mkdir -p "$BUILD_DIR"

# Compile all contracts in CONTRACT_DIR to bytecode using solc
for sol in "$CONTRACT_DIR"/*.sol; do
    echo "Compiling $sol"
    npx --yes solc --bin "$sol" -o "$BUILD_DIR"
    name=$(basename "$sol" .sol)
    prefix="$(basename $(dirname "$CONTRACT_DIR"))_$(basename "$CONTRACT_DIR")_${name}_sol_${name}.bin"
    # solcjs creates files named <dir>_<file>_sol_<contract>.bin
    mv "$BUILD_DIR/$prefix" "$BUILD_DIR/${name}.bin"
    echo
done

# Run echoevm against the compiled contracts with sample calls
run_contract() {
    local bin=$1
    shift
    echo "Executing $bin $*"
    go run -tags evmcli ./cmd/echoevm -bin "$bin" "$@"
    echo
}

run_contract "$BUILD_DIR/Add.bin" -mode full -function 'add(uint256,uint256)' -args '1,2'
run_contract "$BUILD_DIR/Multiply.bin" -mode full -function 'multiply(uint256,uint256)' -args '3,4'
run_contract "$BUILD_DIR/Sum.bin" -mode full -function 'sum(uint256)' -args '5'
