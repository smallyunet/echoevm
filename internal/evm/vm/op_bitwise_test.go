package vm

import (
	"math/big"
	"testing"
)

func TestOpAnd(t *testing.T) {
	i := newInterp()
	i.stack.Push(big.NewInt(3))
	i.stack.Push(big.NewInt(1))
	opAnd(i, 0)
	if i.stack.Pop().Int64() != 1 {
		t.Fatalf("and failed")
	}
}

func TestOpShl(t *testing.T) {
	i := newInterp()
	i.stack.Push(big.NewInt(1))
	i.stack.Push(big.NewInt(1))
	opShl(i, 0)
	if i.stack.Pop().Int64() != 2 {
		t.Fatalf("shl failed")
	}
}
