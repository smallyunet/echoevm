package vm

import (
	"math/big"
	"testing"

	"github.com/smallyunet/echoevm/internal/evm/core"
)

func TestJump(t *testing.T) {
	code := []byte{core.JUMPDEST}
	i := &Interpreter{code: code, stack: core.NewStack(), memory: core.NewMemory()}
	i.stack.PushSafe(big.NewInt(0))
	opJump(i, 0)
	if i.pc != 0 {
		t.Fatalf("jump failed")
	}
}

func TestJumpInvalid(t *testing.T) {
	code := []byte{0x00}
	i := &Interpreter{code: code, stack: core.NewStack(), memory: core.NewMemory()}
	i.stack.PushSafe(big.NewInt(0))
	opJump(i, 0)
	// Now we expect the reverted flag to be set instead of panic
	if !i.reverted {
		t.Fatal("expected reverted flag to be set")
	}
}
