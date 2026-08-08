package vm

import (
	"errors"
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

func TestReturnDataCopyRejectsOutOfBoundsRead(t *testing.T) {
	i := New([]byte{core.PUSH1, 0x03, core.PUSH0, core.PUSH0, core.RETURNDATACOPY, core.STOP}, core.NewMemoryStateDB(), common.Address{})
	i.returnData = []byte{0xaa, 0xbb}
	i.SetGas(100_000)
	i.Run()

	if !errors.Is(i.Err(), ErrReturnDataOutOfBounds) {
		t.Fatalf("error = %v, want return data out of bounds", i.Err())
	}
	if !i.IsReverted() {
		t.Fatal("out-of-bounds RETURNDATACOPY must exceptionally halt")
	}
	if i.Gas() != 0 {
		t.Fatalf("gas = %d, want 0 after exceptional halt", i.Gas())
	}
}

func TestReturnDataCopyAllowsExactBoundary(t *testing.T) {
	i := New([]byte{core.PUSH1, 0x02, core.PUSH0, core.PUSH0, core.RETURNDATACOPY, core.STOP}, core.NewMemoryStateDB(), common.Address{})
	i.returnData = []byte{0xaa, 0xbb}
	i.SetGas(100_000)
	i.Run()

	if i.Err() != nil {
		t.Fatal(i.Err())
	}
	if got := i.Memory().Data(); len(got) < 2 || got[0] != 0xaa || got[1] != 0xbb {
		t.Fatalf("memory = %x, want prefix aabb", got)
	}
}

func TestGas(t *testing.T) {
	i := newInterp()
	opGas(i, 0)
	// Gas now returns a large value since we don't track gas consumption
	if i.stack.PopSafe().Sign() == 0 {
		t.Fatalf("gas should push a non-zero value")
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
