package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestMemoryStateDB(t *testing.T) {
	db := NewMemoryStateDB()
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	// Test Balance
	db.AddBalance(addr, big.NewInt(100))
	if db.GetBalance(addr).Cmp(big.NewInt(100)) != 0 {
		t.Errorf("expected balance 100, got %v", db.GetBalance(addr))
	}
	db.SubBalance(addr, big.NewInt(50))
	if db.GetBalance(addr).Cmp(big.NewInt(50)) != 0 {
		t.Errorf("expected balance 50, got %v", db.GetBalance(addr))
	}

	// Test Nonce
	db.SetNonce(addr, 42)
	if db.GetNonce(addr) != 42 {
		t.Errorf("expected nonce 42, got %v", db.GetNonce(addr))
	}

	// Test Code
	code := []byte{0x01, 0x02, 0x03}
	db.SetCode(addr, code)
	if len(db.GetCode(addr)) != 3 {
		t.Errorf("expected code length 3, got %v", len(db.GetCode(addr)))
	}
	expectedHash := crypto.Keccak256(code)
	if db.GetCodeHash(addr) != common.BytesToHash(expectedHash) {
		t.Errorf("code hash mismatch")
	}

	// Test Storage
	key := common.HexToHash("0x01")
	val := common.HexToHash("0x02")
	db.SetState(addr, key, val)
	if db.GetState(addr, key) != val {
		t.Errorf("expected storage value %v, got %v", val, db.GetState(addr, key))
	}
}
