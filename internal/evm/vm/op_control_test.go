package vm

import (
	"math/big"
	"testing"
)

func TestReturn(t *testing.T) {
	i := newInterp()
	i.memory.Write(0, []byte{1, 2, 3})
	i.stack.Push(big.NewInt(0))
	i.stack.Push(big.NewInt(3))
	opReturn(i, 0)
	if string(i.returned) != "\x01\x02\x03" {
		t.Fatalf("return failed")
	}
}

func TestRevert(t *testing.T) {
	i := newInterp()
	i.memory.Write(0, []byte{4, 5})
	i.stack.Push(big.NewInt(0))
	i.stack.Push(big.NewInt(2))
	opRevert(i, 0)
	if len(i.returned) != 2 || i.returned[0] != 4 {
		t.Fatalf("revert failed")
	}
}
