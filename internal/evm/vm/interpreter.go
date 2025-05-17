package vm

import (
	"fmt"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"math/big"
)

type Interpreter struct {
	code  []byte
	pc    uint64
	stack *core.Stack
}

func New(code []byte) *Interpreter {
	return &Interpreter{code: code, stack: core.NewStack()}
}

func (i *Interpreter) Run() {
	for i.pc < uint64(len(i.code)) {
		op := i.code[i.pc]
		i.pc++

		switch op {
		case core.STOP:
			return

		case core.ADD:
			x, y := i.stack.Pop(), i.stack.Pop()
			i.stack.Push(big.NewInt(0).Add(x, y))

		case core.PUSH1:
			val := big.NewInt(int64(i.code[i.pc]))
			i.pc++
			i.stack.Push(val)

		default:
			panic(fmt.Sprintf("unsupported opcode 0x%02x", op))
		}
	}
}

func (i *Interpreter) Stack() *core.Stack { return i.stack }
