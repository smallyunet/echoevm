package vm

import (
	"github.com/smallyunet/echoevm/internal/evm/core"
	"math/big"
	"testing"
)

func newInterp() *Interpreter {
	return &Interpreter{stack: core.NewStack(), memory: core.NewMemory(), storage: make(map[string]*big.Int)}
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

func TestOpExp(t *testing.T) {
	i := newInterp()
	i.stack.Push(big.NewInt(2)) // base
	i.stack.Push(big.NewInt(3)) // exponent
	opExp(i, 0)
	if i.stack.Pop().Int64() != 8 {
		t.Fatalf("exp failed")
	}
}

func TestOpSgt(t *testing.T) {
	i := newInterp()
	negOne := new(big.Int).Sub(twoTo256, big.NewInt(1))
	i.stack.Push(negOne)        // y = -1
	i.stack.Push(big.NewInt(1)) // x = 1
	opSgt(i, 0)
	if i.stack.Pop().Int64() != 1 {
		t.Fatalf("sgt failed")
	}
}
