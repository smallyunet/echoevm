#!/bin/bash
set -euo pipefail

BUILD_DIR="$(dirname "$0")/build"
CONTRACT_DIR="$(dirname "$0")"
mkdir -p "$BUILD_DIR"

for sol in "$CONTRACT_DIR"/*.sol; do
    echo "Compiling $sol"
    npx --yes solc --bin "$sol" -o "$BUILD_DIR"
done