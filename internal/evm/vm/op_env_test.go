package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func TestCallValue(t *testing.T) {
	i := newInterp()
	opCallValue(i, 0)
	if i.stack.PopSafe().Sign() != 0 {
		t.Fatalf("callvalue not zero")
	}
}

func TestCallDataLoad(t *testing.T) {
	i := NewWithCallData([]byte{core.CALLDATALOAD, core.STOP}, []byte{1, 2, 3}, core.NewMemoryStateDB(), common.Address{})
	i.stack.PushSafe(big.NewInt(0))
	opCallDataLoad(i, 0)
	val := i.stack.PopSafe().Bytes()
	if len(val) != 32 || val[0] != 1 || val[1] != 2 || val[2] != 3 {
		t.Fatalf("calldataload wrong")
	}
}

func TestGas(t *testing.T) {
	i := newInterp()
	opGas(i, 0)
	if i.stack.PopSafe().Sign() != 0 {
		t.Fatalf("gas should push 0")
	}
}

func TestCaller(t *testing.T) {
	i := newInterp()
	opCaller(i, 0)
	if i.stack.PopSafe().Sign() != 0 {
		t.Fatalf("caller should push 0")
	}
}

func TestNumber(t *testing.T) {
	i := newInterp()
	i.SetBlockNumber(123)
	opNumber(i, 0)
	if i.stack.PopSafe().Int64() != 123 {
		t.Fatalf("number wrong")
	}
}
