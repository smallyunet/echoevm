#!/bin/bash
set -euo pipefail

BUILD_DIR="$(dirname "$0")/build"
CONTRACT_DIR="$(dirname "$0")"
mkdir -p "$BUILD_DIR"

echo "Building binary contracts..."
for sol in "$CONTRACT_DIR"/*.sol; do
    if [ -f "$sol" ]; then
        echo "Compiling $sol"
        npx --yes solc --bin "$sol" -o "$BUILD_DIR"
    fi
done

echo "Binary contracts built successfully!"