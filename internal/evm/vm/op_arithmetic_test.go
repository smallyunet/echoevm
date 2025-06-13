package vm

import (
	"github.com/smallyunet/echoevm/internal/evm/core"
	"math/big"
	"testing"
)

func newInterp() *Interpreter {
	return &Interpreter{stack: core.NewStack(), memory: core.NewMemory()}
}

func TestOpAdd(t *testing.T) {
	i := newInterp()
	i.stack.Push(big.NewInt(1))
	i.stack.Push(big.NewInt(2))
	opAdd(i, 0)
	if i.stack.Pop().Int64() != 3 {
		t.Fatalf("add failed")
	}
}

func TestOpDivByZero(t *testing.T) {
	i := newInterp()
	i.stack.Push(big.NewInt(1))
	i.stack.Push(big.NewInt(0))
	opDiv(i, 0)
	if i.stack.Pop().Sign() != 0 {
		t.Fatalf("div by zero should push 0")
	}
}

func TestOpEq(t *testing.T) {
	i := newInterp()
	i.stack.Push(big.NewInt(2))
	i.stack.Push(big.NewInt(2))
	opEq(i, 0)
	if i.stack.Pop().Int64() != 1 {
		t.Fatalf("eq failed")
	}
}
