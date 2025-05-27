// op_stack.go
package vm

import (
	"math/big"
)

func opPush0(i *Interpreter, _ byte) {
	i.stack.Push(big.NewInt(0))
}

func opPop(i *Interpreter, _ byte) {
	i.stack.Pop()
}

func opPush(i *Interpreter, op byte) {
	n := int(op - 0x5f)
	if int(i.pc)+n > len(i.code) {
		panic("invalid PUSH: out of range")
	}
	val := new(big.Int).SetBytes(i.code[i.pc : i.pc+uint64(n)])
	i.pc += uint64(n)
	i.stack.Push(val)
}

func opDup(i *Interpreter, op byte) {
	n := int(op - 0x7f)
	i.stack.Dup(n)
}

func opSwap(i *Interpreter, op byte) {
	n := int(op - 0x8f)
	i.stack.Swap(n)
}
