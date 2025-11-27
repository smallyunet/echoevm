// op_memory.go
package vm

import "math/big"

func opMstore(i *Interpreter, _ byte) {
	offset := i.stack.PopSafe()
	value := i.stack.PopSafe()
	i.memory.Set(offset.Uint64(), value)
}

func opMstore8(i *Interpreter, _ byte) {
	offset := i.stack.PopSafe()
	value := i.stack.PopSafe()
	// MSTORE8 writes the least significant byte
	valByte := byte(value.Uint64() & 0xff)
	i.memory.Write(offset.Uint64(), []byte{valByte})
}

func opMload(i *Interpreter, _ byte) {
	offset := i.stack.PopSafe()
	bytes := i.memory.Get(offset.Uint64())
	i.stack.PushSafe(new(big.Int).SetBytes(bytes))
}

func opCodecopy(i *Interpreter, _ byte) {
	destOffset := i.stack.PopSafe().Uint64()
	codeOffset := i.stack.PopSafe().Uint64()
	size := i.stack.PopSafe().Uint64()

	if codeOffset+size > uint64(len(i.code)) {
		// Instead of panicking, we'll set the reverted flag
		i.reverted = true
		return
	}
	data := i.code[codeOffset : codeOffset+size]
	i.memory.Write(destOffset, data)
}
