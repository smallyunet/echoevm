package core

import (
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestGenesisToStateDB(t *testing.T) {
	// Create a temporary genesis file
	genesisJSON := `
{
  "config": {
    "chainId": 1
  },
  "alloc": {
    "0x1234567890123456789012345678901234567890": {
      "balance": "0x3e8",
      "nonce": 1,
      "code": "0x600160005260206000f3",
      "storage": {
        "0x0000000000000000000000000000000000000000000000000000000000000001": "0x0000000000000000000000000000000000000000000000000000000000000002"
      }
    }
  }
}
`
	tmpfile, err := os.CreateTemp("", "genesis.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(genesisJSON)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Load Genesis
	genesis, err := LoadGenesis(tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to load genesis: %v", err)
	}

	// Apply to StateDB
	db := NewMemoryStateDB()
	if err := genesis.ToStateDB(db); err != nil {
		t.Fatalf("failed to apply genesis: %v", err)
	}

	// Verify
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	if db.GetBalance(addr).Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("expected balance 1000, got %v", db.GetBalance(addr))
	}
	if db.GetNonce(addr) != 1 {
		t.Errorf("expected nonce 1, got %v", db.GetNonce(addr))
	}
	if len(db.GetCode(addr)) != 10 {
		t.Errorf("expected code length 10, got %v", len(db.GetCode(addr)))
	}
	val := db.GetState(addr, common.HexToHash("0x01"))
	if val != common.HexToHash("0x02") {
		t.Errorf("expected storage value 0x02, got %v", val)
	}
}
