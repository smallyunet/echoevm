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
