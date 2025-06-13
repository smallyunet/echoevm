package vm

import (
	"github.com/smallyunet/echoevm/internal/evm/core"
	"testing"
)

func TestInterpreterRunSimple(t *testing.T) {
	code := []byte{core.PUSH1, 0x01, core.PUSH1, 0x02, core.ADD, core.STOP}
	i := New(code)
	i.Run()
	if i.Stack().Len() != 1 {
		t.Fatalf("expected stack len 1, got %d", i.Stack().Len())
	}
	if i.Stack().Pop().Int64() != 3 {
		t.Fatalf("add result wrong")
	}
}

func TestInterpreterCallData(t *testing.T) {
	code := []byte{core.CALLDATASIZE, core.STOP}
	i := NewWithCallData(code, []byte{1, 2, 3})
	i.Run()
	if i.Stack().Pop().Int64() != 3 {
		t.Fatalf("calldatasize wrong")
	}
}
