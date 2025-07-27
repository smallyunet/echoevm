package vm

import (
	"math/big"
	"testing"

	"github.com/smallyunet/echoevm/internal/evm/core"
	"golang.org/x/crypto/sha3"
)

func TestOpSha3(t *testing.T) {
	// Test data: "Hello World"
	testData := []byte("Hello World")

	// Create interpreter with memory and stack
	stack := core.NewStack()
	memory := core.NewMemory()
	interpreter := &Interpreter{
		stack:  stack,
		memory: memory,
	}

	// Write test data to memory at offset 0
	memory.Write(0, testData)

	// Push offset (0) and size (len(testData)) to stack
	stack.Push(big.NewInt(int64(len(testData)))) // size
	stack.Push(big.NewInt(0))                    // offset

	// Execute SHA3 operation
	opSha3(interpreter, core.SHA3)

	// Verify result
	if stack.Len() != 1 {
		t.Fatalf("Expected stack size 1, got %d", stack.Len())
	}

	result := stack.Pop()

	// Compute expected hash using the same method
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(testData)
	expectedHash := hasher.Sum(nil)
	expected := new(big.Int).SetBytes(expectedHash)

	if result.Cmp(expected) != 0 {
		t.Errorf("SHA3 result mismatch.\nExpected: %x\nGot:      %x", expected, result)
	}
}

func TestOpSha3Empty(t *testing.T) {
	// Test empty data
	stack := core.NewStack()
	memory := core.NewMemory()
	interpreter := &Interpreter{
		stack:  stack,
		memory: memory,
	}

	// Push offset (0) and size (0) to stack
	stack.Push(big.NewInt(0)) // size
	stack.Push(big.NewInt(0)) // offset

	// Execute SHA3 operation
	opSha3(interpreter, core.SHA3)

	// Verify result
	if stack.Len() != 1 {
		t.Fatalf("Expected stack size 1, got %d", stack.Len())
	}

	result := stack.Pop()

	// Compute expected hash for empty data
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte{})
	expectedHash := hasher.Sum(nil)
	expected := new(big.Int).SetBytes(expectedHash)

	if result.Cmp(expected) != 0 {
		t.Errorf("SHA3 empty result mismatch.\nExpected: %x\nGot:      %x", expected, result)
	}
}

func TestOpSha3LargeData(t *testing.T) {
	// Test with larger data (256 bytes)
	testData := make([]byte, 256)
	for i := range testData {
		testData[i] = byte(i)
	}

	stack := core.NewStack()
	memory := core.NewMemory()
	interpreter := &Interpreter{
		stack:  stack,
		memory: memory,
	}

	// Write test data to memory at offset 100
	memory.Write(100, testData)

	// Push offset (100) and size (256) to stack
	stack.Push(big.NewInt(256)) // size
	stack.Push(big.NewInt(100)) // offset

	// Execute SHA3 operation
	opSha3(interpreter, core.SHA3)

	// Verify result
	if stack.Len() != 1 {
		t.Fatalf("Expected stack size 1, got %d", stack.Len())
	}

	result := stack.Pop()

	// Compute expected hash
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(testData)
	expectedHash := hasher.Sum(nil)
	expected := new(big.Int).SetBytes(expectedHash)

	if result.Cmp(expected) != 0 {
		t.Errorf("SHA3 large data result mismatch.\nExpected: %x\nGot:      %x", expected, result)
	}
}
