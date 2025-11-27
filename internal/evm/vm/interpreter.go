package vm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

var logger = zerolog.Nop()

// SetLogger allows overriding the package level logger.
func SetLogger(l zerolog.Logger) {
	logger = l
}

type Interpreter struct {
	code        []byte
	pc          uint64
	stack       *core.Stack
	memory      *core.Memory
	calldata    []byte
	returned    []byte
	statedb     core.StateDB
	address     common.Address
	blockNumber uint64
	timestamp   uint64
	coinbase    common.Address
	gasLimit    uint64
	reverted    bool
	logs        []LogEntry
}

// TraceStep captures a single execution step for external tracing.
type TraceStep struct {
	PC         uint64   `json:"pc"`
	Opcode     byte     `json:"opcode"`
	OpcodeName string   `json:"opcode_name"`
	Stack      []string `json:"stack"`
	StackSize  int      `json:"stack_size"`
	Reverted   bool     `json:"reverted"`
	Halt       bool     `json:"halt"`
}

func New(code []byte, statedb core.StateDB, address common.Address) *Interpreter {
	return &Interpreter{
		code:    code,
		stack:   core.NewStack(),
		memory:  core.NewMemory(),
		statedb: statedb,
		address: address,
	}
}

// NewWithCallData creates an interpreter with the provided code and calldata.
func NewWithCallData(code []byte, data []byte, statedb core.StateDB, address common.Address) *Interpreter {
	i := New(code, statedb, address)
	i.calldata = data
	return i
}

// SetCallData sets the calldata that opcodes like CALLDATALOAD operate on.
func (i *Interpreter) SetCallData(data []byte) {
	i.calldata = data
}

// SetBlockNumber sets the block number used by environment opcodes like NUMBER.
func (i *Interpreter) SetBlockNumber(num uint64) {
	i.blockNumber = num
}

func (i *Interpreter) SetTimestamp(ts uint64) {
	i.timestamp = ts
}

func (i *Interpreter) SetCoinbase(addr common.Address) {
	i.coinbase = addr
}

func (i *Interpreter) SetGasLimit(limit uint64) {
	i.gasLimit = limit
}

// Logs returns the collected LOG entries emitted during execution.
func (i *Interpreter) Logs() []LogEntry { return i.logs }

// OpcodeHandler defines a function that executes a specific opcode
type OpcodeHandler func(i *Interpreter, op byte)

// handlerMap maps opcodes to their handlers
var handlerMap = map[byte]OpcodeHandler{}

func init() {
	// arithmetic
	handlerMap[core.ADD] = opAdd
	handlerMap[core.SUB] = opSub
	handlerMap[core.MUL] = opMul
	handlerMap[core.ADDMOD] = opAddmod
	handlerMap[core.MULMOD] = opMulmod
	handlerMap[core.EXP] = opExp
	handlerMap[core.DIV] = opDiv
	handlerMap[core.MOD] = opMod
	handlerMap[core.LT] = opLt
	handlerMap[core.GT] = opGt
	handlerMap[core.SGT] = opSgt
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

	// cryptographic
	handlerMap[core.SHA3] = opSha3

	// memory and code
	handlerMap[core.MSTORE] = opMstore
	handlerMap[core.MSTORE8] = opMstore8
	handlerMap[core.MLOAD] = opMload
	handlerMap[core.CODECOPY] = opCodecopy
	handlerMap[core.SLOAD] = opSload
	handlerMap[core.SSTORE] = opSstore

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
	handlerMap[core.CALLER] = opCaller
	handlerMap[core.CALLDATASIZE] = opCallDataSize
	handlerMap[core.CALLDATALOAD] = opCallDataLoad
	handlerMap[core.CALLDATACOPY] = opCallDataCopy
	handlerMap[core.GAS] = opGas
	handlerMap[core.NUMBER] = opNumber
	handlerMap[core.TIMESTAMP] = opTimestamp
	handlerMap[core.COINBASE] = opCoinbase
	handlerMap[core.GASLIMIT] = opGasLimit

	handlerMap[core.DELEGATECALL] = opDelegateCall

	// logs (LOG0 - LOG4 at 0xa0 - 0xa4)
	for op := byte(0xa0); op <= 0xa4; op++ {
		handlerMap[op] = opLog
	}

	// invalid opcode
	handlerMap[core.INVALID] = opInvalid
}

func (i *Interpreter) Run() {
	for i.pc < uint64(len(i.code)) {
		pc := i.pc
		op := i.code[i.pc]
		i.pc++

		// Log execution step with structured data
		if logger.GetLevel() <= zerolog.TraceLevel {
			logger.Trace().
				Uint64("pc", pc).
				Str("pc_hex", fmt.Sprintf("0x%04x", pc)).
				Uint8("opcode", op).
				Str("opcode_name", core.OpcodeName(op)).
				Int("stack_size", i.stack.Len()).
				Strs("stack", i.stack.Snapshot()).
				Msg("EVM execution step")
		}

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
			// Log invalid opcode error with context
			logger.Error().
				Uint64("pc", pc).
				Uint8("opcode", op).
				Str("opcode_hex", fmt.Sprintf("0x%02x", op)).
				Int("stack_size", i.stack.Len()).
				Strs("stack", i.stack.Snapshot()).
				Msg("Invalid opcode encountered")

			// Instead of panicking, we'll set the reverted flag
			i.reverted = true
			return
		}

		handler(i, op)

		// Log post-execution state
		if logger.GetLevel() <= zerolog.TraceLevel {
			logger.Trace().
				Uint64("pc", i.pc).
				Str("pc_hex", fmt.Sprintf("0x%04x", i.pc)).
				Uint8("opcode", op).
				Str("opcode_name", core.OpcodeName(op)).
				Int("stack_size", i.stack.Len()).
				Strs("stack", i.stack.Snapshot()).
				Msg("EVM execution completed")
		}

		// If RETURN, REVERT or STOP, exit early
		if op == core.RETURN || op == core.REVERT || op == core.STOP {
			return
		}
	}
}

// RunWithHook executes the bytecode emitting a TraceStep to the provided hook before
// and after each opcode. If hook returns false, execution stops early.
func (i *Interpreter) RunWithHook(hook func(step TraceStep) bool) {
	for i.pc < uint64(len(i.code)) {
		pc := i.pc
		op := i.code[i.pc]
		i.pc++

		pre := TraceStep{PC: pc, Opcode: op, OpcodeName: core.OpcodeName(op), Stack: i.stack.Snapshot(), StackSize: i.stack.Len(), Reverted: i.reverted, Halt: false}
		if !hook(pre) {
			return
		}

		if op >= 0x60 && op <= 0x7f { // PUSH1~PUSH32
			opPush(i, op)
		} else if op >= 0x80 && op <= 0x8f { // DUP1~DUP16
			opDup(i, op)
		} else if op >= 0x90 && op <= 0x9f { // SWAP1~SWAP16
			opSwap(i, op)
		} else {
			handler, ok := handlerMap[op]
			if !ok {
				i.reverted = true
				post := TraceStep{PC: i.pc, Opcode: op, OpcodeName: core.OpcodeName(op), Stack: i.stack.Snapshot(), StackSize: i.stack.Len(), Reverted: i.reverted, Halt: true}
				hook(post)
				return
			}
			handler(i, op)
		}

		halt := false
		if op == core.RETURN || op == core.REVERT || op == core.STOP {
			halt = true
		}
		post := TraceStep{PC: i.pc, Opcode: op, OpcodeName: core.OpcodeName(op), Stack: i.stack.Snapshot(), StackSize: i.stack.Len(), Reverted: i.reverted, Halt: halt}
		if !hook(post) {
			return
		}
		if halt {
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

// IsReverted returns true if the execution ended with a REVERT opcode.
func (i *Interpreter) IsReverted() bool {
	return i.reverted
}
