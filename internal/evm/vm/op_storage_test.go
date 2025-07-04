package vm

import (
	"math/big"
	"testing"
)

func TestOpSload(t *testing.T) {
	i := newInterp()
	key := big.NewInt(1)
	i.storage[storageKey(key)] = big.NewInt(42)
	i.stack.Push(key)
	opSload(i, 0)
	if i.stack.Pop().Int64() != 42 {
		t.Fatalf("sload failed")
	}
}

func TestOpSstore(t *testing.T) {
	i := newInterp()
	key := big.NewInt(1)
	i.stack.Push(key)
	i.stack.Push(big.NewInt(7))
	opSstore(i, 0)
	if i.storage[storageKey(key)].Int64() != 7 {
		t.Fatalf("sstore failed")
	}
}
