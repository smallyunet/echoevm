package vm

import (
	"github.com/smallyunet/echoevm/internal/evm/core"
	"math/big"
	"testing"
)

func TestJump(t *testing.T) {
	code := []byte{core.JUMPDEST}
	i := &Interpreter{code: code, stack: core.NewStack(), memory: core.NewMemory()}
	i.stack.Push(big.NewInt(0))
	opJump(i, 0)
	if i.pc != 0 {
		t.Fatalf("jump failed")
	}
}

func TestJumpInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	code := []byte{0x00}
	i := &Interpreter{code: code, stack: core.NewStack(), memory: core.NewMemory()}
	i.stack.Push(big.NewInt(0))
	opJump(i, 0)
}
