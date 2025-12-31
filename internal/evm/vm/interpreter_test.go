package vm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func TestInterpreterRunSimple(t *testing.T) {
	code := []byte{core.PUSH1, 0x01, core.PUSH1, 0x02, core.ADD, core.STOP}
	i := New(code, core.NewMemoryStateDB(), common.Address{})
	i.SetGas(100000) // Set sufficient gas for test
	i.Run()
	if i.Stack().Len() != 1 {
		t.Fatalf("expected stack len 1, got %d", i.Stack().Len())
	}
	if i.Stack().PopSafe().Int64() != 3 {
		t.Fatalf("add result wrong")
	}
}

func TestInterpreterCallData(t *testing.T) {
	code := []byte{core.CALLDATASIZE, core.STOP}
	i := NewWithCallData(code, []byte{1, 2, 3}, core.NewMemoryStateDB(), common.Address{})
	i.SetGas(100000) // Set sufficient gas for test
	i.Run()
	if i.Stack().PopSafe().Int64() != 3 {
		t.Fatalf("calldatasize wrong")
	}
}

func TestInterpreterLoop(t *testing.T) {
	// sum = 0, i = 5
	// loop:
	// if i == 0 goto end
	// sum += i
	// i--
	// goto loop
	// end:

	// Bytecode:
	// PUSH1 0 (sum)
	// PUSH1 5 (i)
	// JUMPDEST (loop start) -> PC 4
	// DUP1 (i)
	// ISZERO
	// PUSH1 19 (jump to end) -> target
	// JUMPI
	// DUP1 (i)
	// SWAP2 (sum is now top, i is 3rd) -> stack: sum, i, i
	// ADD
	// SWAP1 (stack: i, new_sum)
	// PUSH1 1
	// SWAP1
	// SUB (i--)
	// PUSH1 4 (jump to loop)
	// JUMP
	// JUMPDEST (end) -> PC 19
	// POP (pop i)
	// STOP

	code := []byte{
		core.PUSH1, 0,
		core.PUSH1, 5,
		core.JUMPDEST, // 4
		core.DUP1,
		core.ISZERO,
		core.PUSH1, 21,
		core.JUMPI,
		core.DUP1,
		core.SWAP1 + 1, // SWAP2
		core.ADD,
		core.SWAP1,
		core.PUSH1, 1,
		core.SWAP1,
		core.SUB,
		core.PUSH1, 4,
		core.JUMP,
		core.JUMPDEST, // 19
		core.POP,
		core.STOP,
	}

	i := New(code, core.NewMemoryStateDB(), common.Address{})
	i.SetGas(100000)
	i.Run()

	if i.Err() != nil {
		t.Fatalf("execution error: %v", i.Err())
	}

	// sum should be 5+4+3+2+1 = 15
	if i.Stack().Len() != 1 {
		t.Fatalf("expected stack len 1, got %d", i.Stack().Len())
	}
	if i.Stack().PopSafe().Int64() != 15 {
		t.Fatalf("sum wrong, expected 15")
	}
}

func TestInterpreterRevert(t *testing.T) {
	// PUSH1 0xAA
	// PUSH1 0
	// MSTORE
	// PUSH1 1
	// PUSH1 31
	// REVERT

	code := []byte{
		core.PUSH1, 0xAA,
		core.PUSH1, 0,
		core.MSTORE,
		core.PUSH1, 1,
		core.PUSH1, 31,
		core.REVERT,
	}

	i := New(code, core.NewMemoryStateDB(), common.Address{})
	i.SetGas(100000)
	i.Run()

	if !i.IsReverted() {
		t.Fatal("expected reverted state")
	}

	ret := i.ReturnedCode()
	if len(ret) != 1 || ret[0] != 0xAA {
		t.Fatalf("expected return data [0xAA], got %x", ret)
	}
}
