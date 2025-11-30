package vm

import (
	"math/big"
	"testing"
)

func TestOpDelegateCall(t *testing.T) {
	i := newInterp()
	// push dummy arguments: gas, addr, argsOffset, argsLength, retOffset, retLength
	for j := 0; j < 6; j++ {
		i.stack.PushSafe(big.NewInt(int64(j)))
	}
	opDelegateCall(i, 0)
	// With no code to execute, delegatecall should succeed (push 1)
	result := i.stack.PopSafe()
	if result.Sign() != 1 {
		t.Fatalf("delegatecall with empty code should succeed, got %v", result)
	}
}
