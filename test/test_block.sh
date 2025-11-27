#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "Building echoevm..."
go build -o bin/echoevm ./cmd/echoevm

# Create a temporary genesis file
GENESIS_FILE=$(mktemp)
cat > "$GENESIS_FILE" <<EOF
{
  "config": {
    "chainId": 1337
  },
  "alloc": {
    "0x1234567890123456789012345678901234567890": {
      "balance": "0x3e8",
      "nonce": 1,
      "code": "0x600160005260206000f3"
    }
  }
}
EOF

echo "Running block apply..."
OUTPUT=$(./bin/echoevm block apply --genesis "$GENESIS_FILE" 2>&1)

if echo "$OUTPUT" | grep -q "Genesis applied successfully"; then
    echo -e "${GREEN}PASS: Genesis applied successfully${NC}"
else
    echo -e "${RED}FAIL: Genesis application failed${NC}"
    echo "$OUTPUT"
    rm "$GENESIS_FILE"
    exit 1
fi

rm "$GENESIS_FILE"
