package vm

import (
	"fmt"
)

func opInvalid(_ *Interpreter, op byte) {
	panic(fmt.Sprintf("execution hit INVALID opcode: 0x%02x", op))
}
