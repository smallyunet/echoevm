package vm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func TestInterpreterRunSimple(t *testing.T) {
	code := []byte{core.PUSH1, 0x01, core.PUSH1, 0x02, core.ADD, core.STOP}
	i := New(code, core.NewMemoryStateDB(), common.Address{})
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
	i.Run()
	if i.Stack().PopSafe().Int64() != 3 {
		t.Fatalf("calldatasize wrong")
	}
}
