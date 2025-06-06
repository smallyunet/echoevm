package vm

import (
	"fmt"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

type Interpreter struct {
	code     []byte
	pc       uint64
	stack    *core.Stack
	memory   *core.Memory
	returned []byte
}

func New(code []byte) *Interpreter {
	return &Interpreter{
		code:   code,
		stack:  core.NewStack(),
		memory: core.NewMemory(),
	}
}

// OpcodeHandler defines a function that executes a specific opcode
type OpcodeHandler func(i *Interpreter, op byte)

// handlerMap maps opcodes to their handlers
var handlerMap = map[byte]OpcodeHandler{}

func init() {
	// arithmetic
	handlerMap[core.ADD] = opAdd
	handlerMap[core.SUB] = opSub
	handlerMap[core.MUL] = opMul
	handlerMap[core.DIV] = opDiv
	handlerMap[core.MOD] = opMod
	handlerMap[core.EQ] = opEq
	handlerMap[core.ISZERO] = opIsZero

	// memory and code
	handlerMap[core.MSTORE] = opMstore
	handlerMap[core.MLOAD] = opMload
	handlerMap[core.CODECOPY] = opCodecopy

	// stack
	handlerMap[core.POP] = opPop
	handlerMap[core.PUSH0] = opPush0

	// jump
	handlerMap[core.JUMP] = opJump
	handlerMap[core.JUMPI] = opJumpi
	handlerMap[core.JUMPDEST] = opJumpdest

	// control
	handlerMap[core.STOP] = opStop
	handlerMap[core.RETURN] = opReturn

	// environment
	handlerMap[core.CALLVALUE] = opCallValue

	// invalid opcode
	handlerMap[core.INVALID] = opInvalid
}

func (i *Interpreter) Run() {
	for i.pc < uint64(len(i.code)) {
		op := i.code[i.pc]
		i.pc++

		if op >= 0x60 && op <= 0x7f { // PUSH1~PUSH32
			opPush(i, op)
			continue
		}
		if op >= 0x80 && op <= 0x8f { // DUP1~DUP16
			opDup(i, op)
			continue
		}
		if op >= 0x90 && op <= 0x9f { // SWAP1~SWAP16
			opSwap(i, op)
			continue
		}

		handler, ok := handlerMap[op]
		if !ok {
			panic(fmt.Sprintf("unsupported opcode 0x%02x", op))
		}

		handler(i, op)

		// If RETURN or STOP, exit early
		if op == core.RETURN || op == core.STOP {
			return
		}
	}
}

func (i *Interpreter) Stack() *core.Stack {
	return i.stack
}

func (i *Interpreter) Memory() *core.Memory {
	return i.memory
}
