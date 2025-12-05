package vm

import "fmt"

func opInvalid(i *Interpreter, op byte) {
	// INVALID opcode (0xFE) designates the end of execution and consumes all gas.
	// It should return an error to trigger the "consume all gas" behavior in transition.go.
	i.err = fmt.Errorf("invalid opcode: 0x%x", op)
	i.reverted = true
}
