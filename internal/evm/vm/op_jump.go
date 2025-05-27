// op_jump.go
package vm

import (
	"fmt"
)

func opJump(i *Interpreter, _ byte) {
	dst := i.stack.Pop()
	target := dst.Uint64()
	if target >= uint64(len(i.code)) || i.code[target] != 0x5b {
		panic(fmt.Sprintf("invalid jump destination 0x%x", target))
	}
	i.pc = target
}

func opJumpi(i *Interpreter, _ byte) {
	dst := i.stack.Pop()
	cond := i.stack.Pop()
	if cond.Sign() != 0 {
		target := dst.Uint64()
		if target >= uint64(len(i.code)) || i.code[target] != 0x5b {
			panic(fmt.Sprintf("invalid jump destination 0x%x", target))
		}
		i.pc = target
	}
}

func opJumpdest(_ *Interpreter, _ byte) {
	// no-op
}
