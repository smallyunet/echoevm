package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func TestBlobHash_ValidIndex(t *testing.T) {
	statedb := core.NewMemoryStateDB()
	code := []byte{core.BLOBHASH}
	interp := New(code, statedb, common.Address{})
	interp.SetGas(100000)

	// Set up blob hashes
	hash0 := common.HexToHash("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	hash1 := common.HexToHash("0xfedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210")
	interp.SetBlobHashes([]common.Hash{hash0, hash1})

	// Push index 0 onto stack
	interp.stack.PushSafe(big.NewInt(0))
	interp.Run()

	if interp.IsReverted() {
		t.Fatalf("unexpected revert: %v", interp.Err())
	}

	result, err := interp.stack.Pop()
	if err != nil {
		t.Fatalf("failed to pop result: %v", err)
	}

	expected := new(big.Int).SetBytes(hash0.Bytes())
	if result.Cmp(expected) != 0 {
		t.Errorf("BLOBHASH(0) = %s, want %s", result.Text(16), expected.Text(16))
	}
}

func TestBlobHash_SecondIndex(t *testing.T) {
	statedb := core.NewMemoryStateDB()
	code := []byte{core.BLOBHASH}
	interp := New(code, statedb, common.Address{})
	interp.SetGas(100000)

	hash0 := common.HexToHash("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	hash1 := common.HexToHash("0xfedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210")
	interp.SetBlobHashes([]common.Hash{hash0, hash1})

	// Push index 1 onto stack
	interp.stack.PushSafe(big.NewInt(1))
	interp.Run()

	if interp.IsReverted() {
		t.Fatalf("unexpected revert: %v", interp.Err())
	}

	result, err := interp.stack.Pop()
	if err != nil {
		t.Fatalf("failed to pop result: %v", err)
	}

	expected := new(big.Int).SetBytes(hash1.Bytes())
	if result.Cmp(expected) != 0 {
		t.Errorf("BLOBHASH(1) = %s, want %s", result.Text(16), expected.Text(16))
	}
}

func TestBlobHash_OutOfRange(t *testing.T) {
	statedb := core.NewMemoryStateDB()
	code := []byte{core.BLOBHASH}
	interp := New(code, statedb, common.Address{})
	interp.SetGas(100000)

	hash0 := common.HexToHash("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	interp.SetBlobHashes([]common.Hash{hash0})

	// Push index 5 (out of range) onto stack
	interp.stack.PushSafe(big.NewInt(5))
	interp.Run()

	if interp.IsReverted() {
		t.Fatalf("unexpected revert: %v", interp.Err())
	}

	result, err := interp.stack.Pop()
	if err != nil {
		t.Fatalf("failed to pop result: %v", err)
	}

	if result.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("BLOBHASH(5) = %s, want 0 (out of range)", result.Text(16))
	}
}

func TestBlobHash_EmptyList(t *testing.T) {
	statedb := core.NewMemoryStateDB()
	code := []byte{core.BLOBHASH}
	interp := New(code, statedb, common.Address{})
	interp.SetGas(100000)

	// No blob hashes set (empty list)
	interp.SetBlobHashes([]common.Hash{})

	// Push index 0 onto stack
	interp.stack.PushSafe(big.NewInt(0))
	interp.Run()

	if interp.IsReverted() {
		t.Fatalf("unexpected revert: %v", interp.Err())
	}

	result, err := interp.stack.Pop()
	if err != nil {
		t.Fatalf("failed to pop result: %v", err)
	}

	if result.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("BLOBHASH(0) with empty list = %s, want 0", result.Text(16))
	}
}

func TestBlobBaseFee_ReturnsValue(t *testing.T) {
	statedb := core.NewMemoryStateDB()
	code := []byte{core.BLOBBASEFEE}
	interp := New(code, statedb, common.Address{})
	interp.SetGas(100000)

	expectedFee := big.NewInt(1000000000) // 1 gwei
	interp.SetBlobBaseFee(expectedFee)

	interp.Run()

	if interp.IsReverted() {
		t.Fatalf("unexpected revert: %v", interp.Err())
	}

	result, err := interp.stack.Pop()
	if err != nil {
		t.Fatalf("failed to pop result: %v", err)
	}

	if result.Cmp(expectedFee) != 0 {
		t.Errorf("BLOBBASEFEE = %s, want %s", result.Text(10), expectedFee.Text(10))
	}
}

func TestBlobBaseFee_ZeroDefault(t *testing.T) {
	statedb := core.NewMemoryStateDB()
	code := []byte{core.BLOBBASEFEE}
	interp := New(code, statedb, common.Address{})
	interp.SetGas(100000)

	// Do NOT set blobBaseFee - should default to zero

	interp.Run()

	if interp.IsReverted() {
		t.Fatalf("unexpected revert: %v", interp.Err())
	}

	result, err := interp.stack.Pop()
	if err != nil {
		t.Fatalf("failed to pop result: %v", err)
	}

	if result.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("BLOBBASEFEE (unset) = %s, want 0", result.Text(10))
	}
}
