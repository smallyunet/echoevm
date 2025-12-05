// op_jump.go
package vm

import "fmt"

func opJump(i *Interpreter, _ byte) {
	dst := i.stack.PopSafe()
	target := dst.Uint64()
	if target >= uint64(len(i.code)) || i.code[target] != 0x5b {
		i.err = fmt.Errorf("invalid jump destination")
		i.reverted = true
		return
	}
	i.pc = target
}

func opJumpi(i *Interpreter, _ byte) {
	dst := i.stack.PopSafe()
	cond := i.stack.PopSafe()
	if cond.Sign() != 0 {
		target := dst.Uint64()
		if target >= uint64(len(i.code)) || i.code[target] != 0x5b {
			i.err = fmt.Errorf("invalid jump destination")
			i.reverted = true
			return
		}
		i.pc = target
	}
}

func opJumpdest(_ *Interpreter, _ byte) {
	// no-op
}
