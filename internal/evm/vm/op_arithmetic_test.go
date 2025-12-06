package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func newInterp() *Interpreter {
	return &Interpreter{stack: core.NewStack(), memory: core.NewMemory(), statedb: core.NewMemoryStateDB(), address: common.Address{}, gas: 1000000}
}

func TestOpAdd(t *testing.T) {
	i := newInterp()
	i.stack.PushSafe(big.NewInt(1))
	i.stack.PushSafe(big.NewInt(2))
	opAdd(i, 0)
	if i.stack.PopSafe().Int64() != 3 {
		t.Fatalf("add failed")
	}
}

func TestOpDivByZero(t *testing.T) {
	i := newInterp()
	i.stack.PushSafe(big.NewInt(1))
	i.stack.PushSafe(big.NewInt(0))
	opDiv(i, 0)
	if i.stack.PopSafe().Sign() != 0 {
		t.Fatalf("div by zero should push 0")
	}
}

func TestOpEq(t *testing.T) {
	i := newInterp()
	i.stack.PushSafe(big.NewInt(2))
	i.stack.PushSafe(big.NewInt(2))
	opEq(i, 0)
	if i.stack.PopSafe().Int64() != 1 {
		t.Fatalf("eq failed")
	}
}

func TestOpExp(t *testing.T) {
	i := newInterp()
	// EVM stack: first push goes to bottom, pop order is LIFO
	// opExp pops base first, then exponent: base^exp
	// So push exponent first, then base
	i.stack.PushSafe(big.NewInt(3)) // exponent (pushed first, popped second)
	i.stack.PushSafe(big.NewInt(2)) // base (pushed second, popped first)
	opExp(i, 0)
	if i.stack.PopSafe().Int64() != 8 {
		t.Fatalf("exp failed: expected 8 (2^3)")
	}
}

func TestOpSgt(t *testing.T) {
	i := newInterp()
	negOne := new(big.Int).Sub(twoTo256, big.NewInt(1))
	i.stack.PushSafe(negOne)        // y = -1
	i.stack.PushSafe(big.NewInt(1)) // x = 1
	opSgt(i, 0)
	if i.stack.PopSafe().Int64() != 1 {
		t.Fatalf("sgt failed")
	}
}

func TestOpAddMod(t *testing.T) {
	i := newInterp()
	// Stack push order: a, b, m (our op pops m, b, a)
	i.stack.PushSafe(big.NewInt(5))  // a
	i.stack.PushSafe(big.NewInt(7))  // b
	i.stack.PushSafe(big.NewInt(10)) // m
	opAddmod(i, 0)
	if i.stack.Len() != 1 || i.stack.PeekSafe(0).Cmp(big.NewInt(2)) != 0 {
		t.Fatalf("addmod expected 2 got %s", i.stack.PeekSafe(0).String())
	}
	// modulus zero -> 0
	i = newInterp()
	i.stack.PushSafe(big.NewInt(1))
	i.stack.PushSafe(big.NewInt(2))
	i.stack.PushSafe(big.NewInt(0))
	opAddmod(i, 0)
	if i.stack.PeekSafe(0).Sign() != 0 {
		t.Fatalf("addmod modulus 0 expected 0 got %s", i.stack.PeekSafe(0).String())
	}
}

func TestOpMulMod(t *testing.T) {
	i := newInterp()
	i.stack.PushSafe(big.NewInt(5))  // a
	i.stack.PushSafe(big.NewInt(7))  // b
	i.stack.PushSafe(big.NewInt(11)) // m
	opMulmod(i, 0)
	if i.stack.Len() != 1 || i.stack.PeekSafe(0).Cmp(big.NewInt(2)) != 0 {
		t.Fatalf("mulmod expected 2 got %s", i.stack.PeekSafe(0).String())
	}
	// modulus zero -> 0
	i = newInterp()
	i.stack.PushSafe(big.NewInt(3))
	i.stack.PushSafe(big.NewInt(4))
	i.stack.PushSafe(big.NewInt(0))
	opMulmod(i, 0)
	if i.stack.PeekSafe(0).Sign() != 0 {
		t.Fatalf("mulmod modulus 0 expected 0 got %s", i.stack.PeekSafe(0).String())
	}
}
