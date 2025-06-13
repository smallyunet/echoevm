package vm

import (
	"math/big"
	"testing"
)

func TestMstoreLoad(t *testing.T) {
	i := newInterp()
	i.stack.Push(big.NewInt(7)) // value
	i.stack.Push(big.NewInt(0)) // offset
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
	i.stack.Push(big.NewInt(2)) // size
	i.stack.Push(big.NewInt(1)) // offset
	i.stack.Push(big.NewInt(0)) // dest
	opCodecopy(i, 0)
	if b := i.memory.Read(0, 2); b[0] != 2 || b[1] != 3 {
		t.Fatalf("codecopy failed")
	}
}
