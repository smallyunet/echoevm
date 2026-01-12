package vm

import (
	"math/big"
)

// opBlobHash implements EIP-4844 BLOBHASH opcode (0x49).
// Pops an index from the stack and pushes the versioned hash of the blob
// at that index, or zero if the index is out of range.
func opBlobHash(i *Interpreter, op byte) {
	index, err := i.stack.Pop()
	if err != nil {
		i.err = err
		return
	}

	// Check if index is within bounds
	idx := index.Uint64()
	if !index.IsUint64() || idx >= uint64(len(i.blobHashes)) {
		// Out of range: push zero
		i.stack.PushSafe(big.NewInt(0))
		return
	}

	// Push the blob hash at the given index
	hash := i.blobHashes[idx]
	i.stack.PushSafe(new(big.Int).SetBytes(hash.Bytes()))
}

// opBlobBaseFee implements EIP-4844 BLOBBASEFEE opcode (0x4a).
// Pushes the current block's blob base fee onto the stack.
func opBlobBaseFee(i *Interpreter, op byte) {
	if i.blobBaseFee == nil {
		i.stack.PushSafe(big.NewInt(0))
		return
	}
	i.stack.PushSafe(new(big.Int).Set(i.blobBaseFee))
}
