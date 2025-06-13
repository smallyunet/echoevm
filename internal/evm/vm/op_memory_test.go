package vm

import (
	"github.com/smallyunet/echoevm/internal/evm/core"
	"math/big"
	"testing"
)

func TestMstoreLoad(t *testing.T) {
	i := newInterp()
	i.stack.Push(big.NewInt(0))
	i.stack.Push(big.NewInt(7))
	opMstore(i, 0)
	i.stack.Push(big.NewInt(0))
	opMload(i, 0)
	if i.stack.Pop().Int64() != 7 {
		t.Fatalf("mload failed")
	}
}

func TestCodecopy(t *testing.T) {
	i := newInterp()
	i.code = []byte{1, 2, 3, 4}
	i.stack.Push(big.NewInt(0)) // dest
	i.stack.Push(big.NewInt(1)) // offset
	i.stack.Push(big.NewInt(2)) // size
	opCodecopy(i, 0)
	if b := i.memory.Get(0)[:2]; b[0] != 2 || b[1] != 3 {
		t.Fatalf("codecopy failed")
	}
}
