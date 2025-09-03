package vm

import (
	"math/big"
	"testing"
)

func TestOpDelegateCall(t *testing.T) {
	i := newInterp()
	// push dummy arguments
	for j := 0; j < 6; j++ {
		i.stack.PushSafe(big.NewInt(int64(j)))
	}
	opDelegateCall(i, 0)
	if i.stack.PopSafe().Sign() != 0 {
		t.Fatalf("delegatecall should push 0")
	}
}
