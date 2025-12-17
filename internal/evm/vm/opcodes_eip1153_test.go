package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func TestTransientStorage(t *testing.T) {
	// Bytecode:
	// PUSH1 0xCC (value) PUSH1 0x01 (key) TSTORE
	// PUSH1 0x01 (key) TLOAD
	// Expected stack top: 0xCC
	
	code := []byte{
		byte(core.PUSH1), 0xCC,
		byte(core.PUSH1), 0x01,
		byte(core.TSTORE),
		byte(core.PUSH1), 0x01,
		byte(core.TLOAD),
		byte(core.STOP),
	}

	memDB := core.NewMemoryStateDB()
	interpreter := New(code, memDB, common.Address{})
	interpreter.SetGas(100000)
	
	interpreter.Run()
	
	if interpreter.Err() != nil {
		t.Fatalf("Execution failed: %v", interpreter.Err())
	}
	
	stack := interpreter.Stack()
	if stack.Len() != 1 {
		t.Fatalf("Expected stack length 1, got %d", stack.Len())
	}
	
	res, _ := stack.Pop()
	if res.Cmp(big.NewInt(0xCC)) != 0 {
		t.Errorf("Expected 0xCC, got %x", res)
	}
}

func TestTransientStorageIsolation(t *testing.T) {
	// Ensure TLOAD returns 0 for untouched keys
	code := []byte{
		byte(core.PUSH1), 0x02,
		byte(core.TLOAD),
		byte(core.STOP),
	}
	
	memDB := core.NewMemoryStateDB()
	interpreter := New(code, memDB, common.Address{})
	interpreter.SetGas(100000)
	
	interpreter.Run()
	
	if interpreter.Err() != nil {
		t.Fatalf("Execution failed: %v", interpreter.Err())
	}
	
	res, _ := interpreter.Stack().Pop()
	if res == nil {
		t.Fatal("Stack check failed: popped nil")
	}
	if res.Sign() != 0 {
		t.Errorf("Expected 0 for empty transient storage, got %x", res)
	}
}
