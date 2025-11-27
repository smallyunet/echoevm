#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "Building tools..."
go build -o bin/echoevm ./cmd/echoevm
go build -o bin/make_block ./test/make_block

# Create genesis
GENESIS_FILE=$(mktemp)
cat > "$GENESIS_FILE" <<EOF
{
  "config": {
    "chainId": 1337
  },
  "alloc": {
    "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23": {
      "balance": "0x1000000000000000000",
      "nonce": 1
    }
  }
}
EOF
# Sender address matches the private key in make_block/main.go
# 0x2c7536E3605D9C16a7a3D7b1898e529396a65c23

# Create block
BLOCK_FILE=$(mktemp)
./bin/make_block "$BLOCK_FILE"

echo "Running block run..."
./bin/echoevm block run --genesis "$GENESIS_FILE" --block "$BLOCK_FILE"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}PASS: Block executed successfully${NC}"
else
    echo -e "${RED}FAIL: Block execution failed${NC}"
    rm "$GENESIS_FILE" "$BLOCK_FILE"
    exit 1
fi

rm "$GENESIS_FILE" "$BLOCK_FILE"
