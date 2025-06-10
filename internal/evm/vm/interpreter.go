package vm

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

var logger = zerolog.Nop()

// SetLogger allows overriding the package level logger.
func SetLogger(l zerolog.Logger) {
	logger = l
}

type Interpreter struct {
	code     []byte
	pc       uint64
	stack    *core.Stack
	memory   *core.Memory
	calldata []byte
	returned []byte
}

func New(code []byte) *Interpreter {
	return &Interpreter{
		code:   code,
		stack:  core.NewStack(),
		memory: core.NewMemory(),
	}
}

// NewWithCallData creates an interpreter with the provided code and calldata.
func NewWithCallData(code []byte, data []byte) *Interpreter {
	i := New(code)
	i.calldata = data
	return i
}

// SetCallData sets the calldata that opcodes like CALLDATALOAD operate on.
func (i *Interpreter) SetCallData(data []byte) {
	i.calldata = data
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
	handlerMap[core.LT] = opLt
	handlerMap[core.GT] = opGt
	handlerMap[core.SLT] = opSlt
	handlerMap[core.EQ] = opEq
	handlerMap[core.ISZERO] = opIsZero
	handlerMap[core.SIGNEXTEND] = opSignextend

	// bitwise and shift
	handlerMap[core.AND] = opAnd
	handlerMap[core.OR] = opOr
	handlerMap[core.XOR] = opXor
	handlerMap[core.NOT] = opNot
	handlerMap[core.BYTE] = opByte
	handlerMap[core.SHL] = opShl
	handlerMap[core.SHR] = opShr
	handlerMap[core.SAR] = opSar

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
	handlerMap[core.REVERT] = opRevert

	// environment
	handlerMap[core.CALLVALUE] = opCallValue
	handlerMap[core.CALLDATASIZE] = opCallDataSize
	handlerMap[core.CALLDATALOAD] = opCallDataLoad
	handlerMap[core.CALLDATACOPY] = opCallDataCopy

	// invalid opcode
	handlerMap[core.INVALID] = opInvalid
}

func (i *Interpreter) Run() {
	for i.pc < uint64(len(i.code)) {
		pc := i.pc
		op := i.code[i.pc]
		i.pc++
		logger.Trace().
			Str("pc", fmt.Sprintf("0x%04x", pc)).
			Str("op", core.OpcodeName(op)).
			Int("stack", i.stack.Len()).
			Msg("step")

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
		logger.Trace().
			Str("pc", fmt.Sprintf("0x%04x", i.pc)).
			Str("op", core.OpcodeName(op)).
			Any("stack", i.stack.Snapshot()).
			Msg("after")

		// If RETURN, REVERT or STOP, exit early
		if op == core.RETURN || op == core.REVERT || op == core.STOP {
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

// ReturnedCode returns the byte slice produced by a RETURN opcode.
// It is primarily used to obtain the runtime bytecode generated during
// contract creation.
func (i *Interpreter) ReturnedCode() []byte {
	return i.returned
}
