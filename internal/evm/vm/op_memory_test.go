package vm

import (
	"math/big"
	"strings"
	"testing"
)

func TestMstoreLoad(t *testing.T) {
	i := newInterp()
	i.stack.PushSafe(big.NewInt(7)) // value
	i.stack.PushSafe(big.NewInt(0)) // offset
	opMstore(i, 0)
	i.stack.PushSafe(big.NewInt(0))
	opMload(i, 0)
	if i.stack.PopSafe().Int64() != 7 {
		t.Fatalf("mload failed")
	}
}

func TestCodecopy(t *testing.T) {
	i := newInterp()
	i.code = []byte{1, 2, 3, 4}
	i.stack.PushSafe(big.NewInt(2)) // size
	i.stack.PushSafe(big.NewInt(1)) // offset
	i.stack.PushSafe(big.NewInt(0)) // dest
	opCodecopy(i, 0)
	if b := i.memory.Read(0, 2); b[0] != 2 || b[1] != 3 {
		t.Fatalf("codecopy failed")
	}
}

func TestFixedMemoryOpcodesRejectOffsetOverflow(t *testing.T) {
	overflow := new(big.Int).Lsh(big.NewInt(1), 64)
	tests := []struct {
		name string
		op   func(*Interpreter, byte)
		push func(*Interpreter)
	}{
		{name: "mload", op: opMload, push: func(i *Interpreter) { i.stack.PushSafe(overflow) }},
		{name: "mstore", op: opMstore, push: func(i *Interpreter) {
			i.stack.PushSafe(big.NewInt(1))
			i.stack.PushSafe(overflow)
		}},
		{name: "mstore8", op: opMstore8, push: func(i *Interpreter) {
			i.stack.PushSafe(big.NewInt(1))
			i.stack.PushSafe(overflow)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := newInterp()
			tt.push(i)
			tt.op(i, 0)
			if i.Err() == nil || !i.IsReverted() {
				t.Fatalf("error = %v, reverted = %t; want memory-expansion fault", i.Err(), i.IsReverted())
			}
			if !strings.Contains(i.Err().Error(), "memory expansion") {
				t.Fatalf("error = %v, want memory-expansion fault", i.Err())
			}
		})
	}
}
