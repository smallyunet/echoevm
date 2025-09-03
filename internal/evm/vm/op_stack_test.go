package vm

import (
	"github.com/smallyunet/echoevm/internal/evm/core"
	"math/big"
	"testing"
)

func TestOpPush0Pop(t *testing.T) {
	i := newInterp()
	opPush0(i, 0)
	if i.stack.PopSafe().Int64() != 0 {
		t.Fatalf("push0 failed")
	}
}

func TestOpPushDupSwap(t *testing.T) {
	i := newInterp()
	i.code = []byte{core.PUSH1, 0x01}
	i.pc = 1
	opPush(i, core.PUSH1)
	if i.stack.PopSafe().Int64() != 1 {
		t.Fatalf("push1 failed")
	}
	// test DUP1
	i.stack.PushSafe(big.NewInt(1))
	opDup(i, core.DUP1)
	if i.stack.PeekSafe(0).Int64() != 1 {
		t.Fatalf("dup failed")
	}
	// test SWAP1
	i.stack.PushSafe(big.NewInt(2))
	opSwap(i, core.SWAP1)
	if i.stack.PeekSafe(0).Int64() != 1 {
		t.Fatalf("swap failed")
	}
}
