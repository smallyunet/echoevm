package vm

import (
	"fmt"
	"math/big"

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
	caller      common.Address
	origin      common.Address
	callvalue   *big.Int
	blockNumber uint64
	timestamp   uint64
	coinbase    common.Address
	gasLimit    uint64
	gas         uint64 // Remaining gas
	maxMemorySize uint64 // Highest memory size (in bytes) paid for
	gasPrice    *big.Int
	chainID     *big.Int
	baseFee     *big.Int
	difficulty  *big.Int
	random      *big.Int // PREVRANDAO value for post-merge (used by DIFFICULTY opcode)
	reverted    bool
	err         error
	logs        []LogEntry
	returnData  []byte // return data from last CALL
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
	IsPost     bool     `json:"is_post"`
}

func New(code []byte, statedb core.StateDB, address common.Address) *Interpreter {
	return &Interpreter{
		code:       code,
		stack:      core.NewStack(),
		memory:     core.NewMemory(),
		statedb:    statedb,
		address:    address,
		callvalue:  big.NewInt(0),
		gasPrice:   big.NewInt(0),
		chainID:    big.NewInt(1),
		baseFee:    big.NewInt(0),
		difficulty: big.NewInt(0),
		gas:        0,
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

func (i *Interpreter) SetBlockGasLimit(limit uint64) {
	i.gasLimit = limit
}

func (i *Interpreter) SetGas(gas uint64) {
	i.gas = gas
}

// Gas returns the remaining gas.
func (i *Interpreter) Gas() uint64 {
	return i.gas
}

func (i *Interpreter) consumeMemoryExpansion(offset, size uint64) bool {
	if size == 0 {
		return true
	}
	newSize := offset + size
	if newSize <= i.maxMemorySize {
		return true
	}
	
	oldCost := core.MemoryGasCost(i.maxMemorySize)
	newCost := core.MemoryGasCost(newSize)
	cost := newCost - oldCost
	
	if i.gas < cost {
		i.err = fmt.Errorf("out of gas: memory expansion")
		i.reverted = true
		return false
	}
	i.gas -= cost
	i.maxMemorySize = (newSize + 31) / 32 * 32
	return true
}

func (i *Interpreter) SetCaller(addr common.Address) {
	i.caller = addr
}

func (i *Interpreter) SetOrigin(addr common.Address) {
	i.origin = addr
}

func (i *Interpreter) SetCallValue(val *big.Int) {
	i.callvalue = val
}

func (i *Interpreter) SetGasPrice(price *big.Int) {
	i.gasPrice = price
}

func (i *Interpreter) SetChainID(id *big.Int) {
	i.chainID = id
}

func (i *Interpreter) SetBaseFee(fee *big.Int) {
	i.baseFee = fee
}

func (i *Interpreter) SetDifficulty(diff *big.Int) {
	i.difficulty = diff
}

// SetRandom sets the PREVRANDAO value for post-merge blocks.
// The DIFFICULTY opcode returns this value after The Merge.
func (i *Interpreter) SetRandom(random *big.Int) {
	i.random = random
}

// Logs returns the collected LOG entries emitted during execution.
func (i *Interpreter) Logs() []LogEntry { return i.logs }

// OpcodeHandler defines a function that executes a specific opcode
type OpcodeHandler func(i *Interpreter, op byte)

// handlerMap maps opcodes to their handlers
var handlerMap = [256]OpcodeHandler{}

func init() {
	// arithmetic
	handlerMap[core.ADD] = opAdd
	handlerMap[core.SUB] = opSub
	handlerMap[core.MUL] = opMul
	handlerMap[core.ADDMOD] = opAddmod
	handlerMap[core.MULMOD] = opMulmod
	handlerMap[core.EXP] = opExp
	handlerMap[core.DIV] = opDiv
	handlerMap[core.SDIV] = opSdiv
	handlerMap[core.MOD] = opMod
	handlerMap[core.SMOD] = opSmod
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
	handlerMap[core.ADDRESS] = opAddress
	handlerMap[core.BALANCE] = opBalance
	handlerMap[core.ORIGIN] = opOrigin
	handlerMap[core.CALLVALUE] = opCallValue
	handlerMap[core.CALLER] = opCaller
	handlerMap[core.CALLDATASIZE] = opCallDataSize
	handlerMap[core.CALLDATALOAD] = opCallDataLoad
	handlerMap[core.CALLDATACOPY] = opCallDataCopy
	handlerMap[core.CODESIZE] = opCodeSize
	handlerMap[core.GASPRICE] = opGasPrice
	handlerMap[core.EXTCODESIZE] = opExtCodeSize
	handlerMap[core.EXTCODECOPY] = opExtCodeCopy
	handlerMap[core.RETURNDATASIZE] = opReturnDataSize
	handlerMap[core.RETURNDATACOPY] = opReturnDataCopy
	handlerMap[core.EXTCODEHASH] = opExtCodeHash
	handlerMap[core.BLOCKHASH] = opBlockHash
	handlerMap[core.COINBASE] = opCoinbase
	handlerMap[core.TIMESTAMP] = opTimestamp
	handlerMap[core.NUMBER] = opNumber
	handlerMap[core.DIFFICULTY] = opDifficulty
	handlerMap[core.GASLIMIT] = opGasLimit
	handlerMap[core.CHAINID] = opChainID
	handlerMap[core.SELFBALANCE] = opSelfBalance
	handlerMap[core.BASEFEE] = opBaseFee
	handlerMap[core.PC] = opPC
	handlerMap[core.MSIZE] = opMSize
	handlerMap[core.GAS] = opGas

	// call operations
	handlerMap[core.CREATE] = opCreate
	handlerMap[core.CALL] = opCall
	handlerMap[core.CALLCODE] = opCallCode
	handlerMap[core.DELEGATECALL] = opDelegateCall
	handlerMap[core.CREATE2] = opCreate2
	handlerMap[core.STATICCALL] = opStaticCall

	// logs (LOG0 - LOG4 at 0xa0 - 0xa4)
	for op := byte(0xa0); op <= 0xa4; op++ {
		handlerMap[op] = opLog
	}

	// self destruct
	handlerMap[core.SELFDESTRUCT] = opSelfDestruct

	// invalid opcode
	handlerMap[core.INVALID] = opInvalid

	// PUSH, DUP, SWAP
	for i := 0; i < 32; i++ {
		handlerMap[core.PUSH1+byte(i)] = opPush
	}
	for i := 0; i < 16; i++ {
		handlerMap[core.DUP1+byte(i)] = opDup
		handlerMap[core.SWAP1+byte(i)] = opSwap
	}
}

func (i *Interpreter) Run() {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				i.err = err
			} else {
				i.err = fmt.Errorf("execution panic: %v", r)
			}
			i.reverted = true
			logger.Error().Err(i.err).Msg("EVM execution recovered from panic")
		}
	}()

	for i.pc < uint64(len(i.code)) {
		pc := i.pc
		op := i.code[i.pc]
		i.pc++

		// Gas deduction
		cost := core.GasTable[op]

		if i.gas < cost {
			i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, cost)
			i.reverted = true
			return
		}
		i.gas -= cost

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

		handler := handlerMap[op]
		if handler == nil {
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

		// Check for errors or revert
		if i.err != nil {
			i.gas = 0
			return
		}
		if i.reverted {
			return
		}

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
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				i.err = err
			} else {
				i.err = fmt.Errorf("execution panic: %v", r)
			}
			i.reverted = true
			// Emit a final trace step for the error if possible?
			// For now just return, caller can check i.Err()
		}
	}()

	for i.pc < uint64(len(i.code)) {
		pc := i.pc
		op := i.code[i.pc]
		i.pc++

		pre := TraceStep{PC: pc, Opcode: op, OpcodeName: core.OpcodeName(op), Stack: i.stack.Snapshot(), StackSize: i.stack.Len(), Reverted: i.reverted, Halt: false, IsPost: false}
		if !hook(pre) {
			return
		}

		handler := handlerMap[op]
		if handler == nil {
			i.reverted = true
			post := TraceStep{PC: i.pc, Opcode: op, OpcodeName: core.OpcodeName(op), Stack: i.stack.Snapshot(), StackSize: i.stack.Len(), Reverted: i.reverted, Halt: true, IsPost: true}
			hook(post)
			return
		}
		handler(i, op)

		halt := false
		if op == core.RETURN || op == core.REVERT || op == core.STOP {
			halt = true
		}
		post := TraceStep{PC: i.pc, Opcode: op, OpcodeName: core.OpcodeName(op), Stack: i.stack.Snapshot(), StackSize: i.stack.Len(), Reverted: i.reverted, Halt: halt, IsPost: true}
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

func (i *Interpreter) IsReverted() bool {
	return i.reverted
}

func (i *Interpreter) Err() error {
	return i.err
}

func (i *Interpreter) SetStack(s *core.Stack) {
	i.stack = s
}

func (i *Interpreter) SetMemory(m *core.Memory) {
	i.memory = m
}
