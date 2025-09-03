// op_sha3.go
package vm

import (
	"math/big"

	"golang.org/x/crypto/sha3"
)

// opSha3 implements the SHA3 (KECCAK256) opcode
// Pops offset and size from stack, reads memory data, computes Keccak256 hash, and pushes result
func opSha3(i *Interpreter, _ byte) {
	// Pop offset and size from the stack
	offset := i.stack.PopSafe()
	size := i.stack.PopSafe()

	// Get data from memory
	data := i.memory.Read(offset.Uint64(), size.Uint64())

	// Compute Keccak256 hash
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)
	hash := hasher.Sum(nil)

	// Push the hash result onto the stack as a big.Int
	result := new(big.Int).SetBytes(hash)
	i.stack.PushSafe(result)
}
