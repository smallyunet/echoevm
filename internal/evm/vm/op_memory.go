// op_memory.go
package vm

import "math/big"

func opMstore(i *Interpreter, _ byte) {
	offset := i.stack.Pop()
	value := i.stack.Pop()
	i.memory.Set(offset.Uint64(), value)
}

func opMload(i *Interpreter, _ byte) {
	offset := i.stack.Pop()
	bytes := i.memory.Get(offset.Uint64())
	i.stack.Push(new(big.Int).SetBytes(bytes))
}

func opCodecopy(i *Interpreter, _ byte) {
	destOffset := i.stack.Pop().Uint64()
	codeOffset := i.stack.Pop().Uint64()
	size := i.stack.Pop().Uint64()

	if codeOffset+size > uint64(len(i.code)) {
		panic("CODECOPY out of range")
	}
	data := i.code[codeOffset : codeOffset+size]
	i.memory.Write(destOffset, data)
}
