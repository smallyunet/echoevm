// op_memory.go
package vm

import (
	"fmt"
	"math/big"
)

func opMstore(i *Interpreter, _ byte) {
	offset := i.stack.PopSafe()
	value := i.stack.PopSafe()
	if !i.consumeMemoryExpansion(offset.Uint64(), 32) {
		return
	}
	i.memory.Set(offset.Uint64(), value)
}

func opMstore8(i *Interpreter, _ byte) {
	offset := i.stack.PopSafe()
	value := i.stack.PopSafe()
	if !i.consumeMemoryExpansion(offset.Uint64(), 1) {
		return
	}
	// MSTORE8 writes the least significant byte
	valByte := byte(value.Uint64() & 0xff)
	i.memory.Write(offset.Uint64(), []byte{valByte})
}

func opMload(i *Interpreter, _ byte) {
	offset := i.stack.PopSafe()
	if !i.consumeMemoryExpansion(offset.Uint64(), 32) {
		return
	}
	bytes := i.memory.Get(offset.Uint64())
	i.stack.PushSafe(new(big.Int).SetBytes(bytes))
}

func opCodecopy(i *Interpreter, _ byte) {
	destOffset := i.stack.PopSafe().Uint64()
	codeOffset := i.stack.PopSafe().Uint64()
	size := i.stack.PopSafe().Uint64()

	if !i.consumeMemoryExpansion(destOffset, size) {
		return
	}

	// Dynamic gas: 3 * words
	words := (size + 31) / 32
	copyCost := words * 3
	if i.gas < copyCost {
		i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, copyCost)
		i.reverted = true
		return
	}
	i.gas -= copyCost

	var data []byte
	if codeOffset >= uint64(len(i.code)) {
		data = make([]byte, size) // All zeros
	} else {
		end := codeOffset + size
		if end > uint64(len(i.code)) {
			end = uint64(len(i.code))
		}
		data = make([]byte, size)
		copy(data, i.code[codeOffset:end])
	}
	i.memory.Write(destOffset, data)
}
