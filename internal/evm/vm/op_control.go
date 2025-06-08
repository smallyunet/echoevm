// op_control.go
package vm

import (
	"fmt"
)

func opStop(_ *Interpreter, _ byte) {
	// halt execution
}

func opReturn(i *Interpreter, _ byte) {
	offset := i.stack.Pop().Uint64()
	size := i.stack.Pop().Uint64()
	ret := i.memory.Get(offset)[:size]
	i.returned = ret
	fmt.Printf("RETURN: 0x%x\n", ret)
}

// opRevert halts execution and marks the returned data as an error payload.
// This simplified EVM does not differentiate between revert and return beyond
// printing a message and storing the payload.
func opRevert(i *Interpreter, _ byte) {
	offset := i.stack.Pop().Uint64()
	size := i.stack.Pop().Uint64()
	ret := i.memory.Get(offset)[:size]
	i.returned = ret
	fmt.Printf("REVERT: 0x%x\n", ret)
}
