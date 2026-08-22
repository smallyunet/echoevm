package vm

import (
	"math/big"
	"testing"

	"github.com/smallyunet/echoevm/internal/eth/common"
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
	i.SetGas(100000)
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

func TestOpSstoreWarmGasCost(t *testing.T) {
	tests := []struct {
		name       string
		original   byte
		value      byte
		wantOpCost uint64
	}{
		{name: "set empty slot", original: 0x00, value: 0x42, wantOpCost: 20_000},
		{name: "update clean slot", original: 0x01, value: 0x42, wantOpCost: 2_900},
		{name: "unchanged slot", original: 0x42, value: 0x42, wantOpCost: 100},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			address := common.Address{}
			key := common.BigToHash(big.NewInt(1))
			db := core.NewMemoryStateDB()
			db.InitState(address, key, common.BytesToHash([]byte{test.original}))
			db.PrepareTransaction()
			db.AddSlotToAccessList(address, key)

			code := []byte{core.PUSH1, test.value, core.PUSH1, 0x01, core.SSTORE, core.STOP}
			interpreter := New(code, db, address)
			interpreter.SetGas(100_000)
			interpreter.Run()
			if interpreter.Err() != nil {
				t.Fatal(interpreter.Err())
			}
			const pushCost = uint64(6)
			if got := uint64(100_000) - interpreter.Gas(); got != pushCost+test.wantOpCost {
				t.Fatalf("gas used = %d, want %d", got, pushCost+test.wantOpCost)
			}
		})
	}
}
