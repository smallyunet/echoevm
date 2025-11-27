package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func TestOpSstoreSload(t *testing.T) {
	code := []byte{
		core.PUSH1, 0x42, // value
		core.PUSH1, 0x01, // key
		core.SSTORE,
		core.PUSH1, 0x01, // key
		core.SLOAD,
	}
	db := core.NewMemoryStateDB()
	i := New(code, db, common.Address{})
	i.Run()

	if i.Stack().Len() != 1 {
		t.Fatalf("expected stack len 1, got %d", i.Stack().Len())
	}
	val := i.Stack().PopSafe()
	if val.Int64() != 0x42 {
		t.Errorf("expected 0x42, got %v", val)
	}

	// Verify directly in StateDB
	stored := db.GetState(common.Address{}, common.BigToHash(big.NewInt(1)))
	if stored != common.BigToHash(big.NewInt(0x42)) {
		t.Errorf("expected storage 0x42, got %v", stored)
	}
}
