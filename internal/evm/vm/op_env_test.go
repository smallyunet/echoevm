package vm

import (
	"github.com/smallyunet/echoevm/internal/evm/core"
	"math/big"
	"testing"
)

func TestCallValue(t *testing.T) {
	i := newInterp()
	opCallValue(i, 0)
	if i.stack.Pop().Sign() != 0 {
		t.Fatalf("callvalue not zero")
	}
}

func TestCallDataLoad(t *testing.T) {
	i := NewWithCallData([]byte{core.CALLDATALOAD, core.STOP}, []byte{1, 2, 3})
	i.stack.Push(big.NewInt(0))
	opCallDataLoad(i, 0)
	val := i.stack.Pop().Bytes()
	if len(val) != 32 || val[0] != 1 || val[1] != 2 || val[2] != 3 {
		t.Fatalf("calldataload wrong")
	}
}

func TestGas(t *testing.T) {
	i := newInterp()
	opGas(i, 0)
	if i.stack.Pop().Sign() != 0 {
		t.Fatalf("gas should push 0")
	}
}
