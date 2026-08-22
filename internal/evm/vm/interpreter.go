package vm

import (
	"fmt"
	"math"
	"math/big"
	"math/bits"

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
	code          []byte
	pc            uint64
	stack         *core.Stack
	memory        *core.Memory
	calldata      []byte
	returned      []byte
	statedb       core.StateDB
	address       common.Address
	caller        common.Address
	origin        common.Address
	callvalue     *big.Int
	blockNumber   uint64
	timestamp     uint64
	coinbase      common.Address
	gasLimit      uint64
	gas           uint64 // Remaining gas
	maxMemorySize uint64 // Highest memory size (in bytes) paid for
	gasPrice      *big.Int
	chainID       *big.Int
	baseFee       *big.Int
	difficulty    *big.Int
	random        *big.Int          // PREVRANDAO value for post-merge (used by DIFFICULTY opcode)
	chainConfig   *core.ChainConfig // Fork configuration
	reverted      bool
	err           error
	logs          []LogEntry
	returnData    []byte        // return data from last CALL
	blobHashes    []common.Hash // EIP-4844: versioned hashes of transaction blobs
	blobBaseFee   *big.Int      // EIP-4844: blob base fee for the block
	traceHook     func(TraceStep) bool
	traceDepth    int
	traceDetails  bool
	readOnly      bool
}

// TraceStep captures a single execution step for external tracing.
type TraceStep struct {
	PC         uint64               `json:"pc"`
	Opcode     byte                 `json:"opcode"`
	OpcodeName string               `json:"opcode_name"`
	Stack      []string             `json:"stack"`
	StackSize  int                  `json:"stack_size"`
	Gas        uint64               `json:"gas"`
	Reverted   bool                 `json:"reverted"`
	Halt       bool                 `json:"halt"`
	IsPost     bool                 `json:"is_post"`
	Depth      int                  `json:"depth"`
	Address    string               `json:"address"`
	Error      string               `json:"error,omitempty"`
	Memory     []byte               `json:"-"`
	Storage    []TraceStorageAccess `json:"-"`
}

// TraceStorageAccess captures the storage context visible before an opcode.
// It is intentionally part of the internal trace hook rather than StateDB so
// normal execution does not pay for a separate state-observer abstraction.
type TraceStorageAccess struct {
	Kind      string
	Address   string
	Slot      string
	Before    string
	After     string
	Original  string
	Warm      bool
	Transient bool
}

func New(code []byte, statedb core.StateDB, address common.Address) *Interpreter {
	return &Interpreter{
		code:        code,
		stack:       core.NewStack(),
		memory:      core.NewMemory(),
		statedb:     statedb,
		address:     address,
		callvalue:   big.NewInt(0),
		gasPrice:    big.NewInt(0),
		chainID:     big.NewInt(1),
		baseFee:     big.NewInt(0),
		difficulty:  big.NewInt(0),
		gas:         0,
		chainConfig: core.DefaultChainConfig,
	}
}

// NewWithCallData creates an interpreter with the provided code and calldata.
func NewWithCallData(code []byte, data []byte, statedb core.StateDB, address common.Address) *Interpreter {
	i := New(code, statedb, address)
	i.calldata = data
	return i
}

// SetChainConfig sets the chain configuration.
func (i *Interpreter) SetChainConfig(cfg *core.ChainConfig) {
	i.chainConfig = cfg
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

// SetReadOnly marks the current call frame as static. Child frames inherit the
// flag so CALL, CALLCODE, and DELEGATECALL cannot escape STATICCALL protection.
func (i *Interpreter) SetReadOnly(readOnly bool) {
	i.readOnly = readOnly
}

func (i *Interpreter) rejectWriteProtection() bool {
	if !i.readOnly {
		return false
	}
	i.err = ErrWriteProtection
	i.reverted = true
	return true
}

// Gas returns the remaining gas.
func (i *Interpreter) Gas() uint64 {
	return i.gas
}

func (i *Interpreter) consumeMemoryExpansion(offset, size uint64) bool {
	if size == 0 {
		return true
	}
	if offset > math.MaxUint64-size {
		return i.failMemoryExpansion("offset overflow")
	}
	newSize := offset + size
	if newSize <= i.maxMemorySize {
		return true
	}
	if newSize > math.MaxUint64-31 {
		return i.failMemoryExpansion("size overflow")
	}
	newSize = (newSize + 31) / 32 * 32

	oldCost, oldOverflow := memoryGasCost(i.maxMemorySize)
	newCost, newOverflow := memoryGasCost(newSize)
	if oldOverflow || newOverflow || newCost < oldCost {
		return i.failMemoryExpansion("gas overflow")
	}
	cost := newCost - oldCost

	if i.gas < cost {
		i.err = fmt.Errorf("out of gas: memory expansion")
		i.reverted = true
		return false
	}
	i.gas -= cost
	i.maxMemorySize = newSize
	return true
}

func (i *Interpreter) consumeFixedMemoryExpansion(offset *big.Int, size uint64) (uint64, bool) {
	if !offset.IsUint64() {
		return 0, i.failMemoryExpansion("offset overflow")
	}
	offset64 := offset.Uint64()
	return offset64, i.consumeMemoryExpansion(offset64, size)
}

func (i *Interpreter) failMemoryExpansion(reason string) bool {
	i.err = fmt.Errorf("out of gas: memory expansion %s", reason)
	i.reverted = true
	return false
}

func memoryGasCost(size uint64) (uint64, bool) {
	words := size / 32
	hi, lo := bits.Mul64(words, words)
	if hi >= 512 {
		return 0, true
	}
	quadratic, _ := bits.Div64(hi, lo, 512)
	linearHi, linear := bits.Mul64(words, core.GasMemory)
	if linearHi != 0 {
		return 0, true
	}
	cost, carry := bits.Add64(quadratic, linear, 0)
	return cost, carry != 0
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

// SetBlobHashes sets the versioned blob hashes for EIP-4844 transactions.
func (i *Interpreter) SetBlobHashes(hashes []common.Hash) {
	i.blobHashes = hashes
}

// SetBlobBaseFee sets the blob base fee for the current block (EIP-4844).
func (i *Interpreter) SetBlobBaseFee(fee *big.Int) {
	i.blobBaseFee = fee
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
	handlerMap[core.CLZ] = opClz

	// cryptographic
	handlerMap[core.SHA3] = opSha3

	// memory and code
	handlerMap[core.MSTORE] = opMstore
	handlerMap[core.MSTORE8] = opMstore8
	handlerMap[core.MLOAD] = opMload
	handlerMap[core.CODECOPY] = opCodecopy
	handlerMap[core.SLOAD] = opSload
	handlerMap[core.SSTORE] = opSstore
	handlerMap[core.TLOAD] = opTload
	handlerMap[core.TSTORE] = opTstore
	handlerMap[core.MCOPY] = opMcopy

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
	handlerMap[core.BLOBHASH] = opBlobHash       // EIP-4844
	handlerMap[core.BLOBBASEFEE] = opBlobBaseFee // EIP-4844
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
	i.run(i.traceHook)
}

// RunWithHook executes bytecode using the same execution loop as Run and emits
// a TraceStep before and after each opcode. Returning false from the hook stops
// execution without executing another opcode.
func (i *Interpreter) RunWithHook(hook func(step TraceStep) bool) {
	i.traceHook = hook
	i.run(hook)
}

// SetTraceContext installs a transaction-wide trace hook. Child interpreters
// inherit it with an incremented depth so nested calls remain observable.
func (i *Interpreter) SetTraceContext(hook func(step TraceStep) bool, depth int) {
	i.traceHook = hook
	i.traceDepth = depth
}

// SetTraceDetails enables memory snapshots and storage access context for
// explainable traces. Lightweight conformance hooks can leave it disabled.
func (i *Interpreter) SetTraceDetails(enabled bool) {
	i.traceDetails = enabled
}

func (i *Interpreter) inheritExecutionContext(parent *Interpreter) {
	i.blockNumber = parent.blockNumber
	i.timestamp = parent.timestamp
	i.coinbase = parent.coinbase
	i.gasLimit = parent.gasLimit
	i.origin = parent.origin
	if parent.gasPrice != nil {
		i.gasPrice = new(big.Int).Set(parent.gasPrice)
	}
	if parent.chainID != nil {
		i.chainID = new(big.Int).Set(parent.chainID)
	}
	if parent.baseFee != nil {
		i.baseFee = new(big.Int).Set(parent.baseFee)
	}
	if parent.difficulty != nil {
		i.difficulty = new(big.Int).Set(parent.difficulty)
	}
	if parent.random != nil {
		i.random = new(big.Int).Set(parent.random)
	}
	i.chainConfig = parent.chainConfig
	i.blobHashes = append([]common.Hash(nil), parent.blobHashes...)
	if parent.blobBaseFee != nil {
		i.blobBaseFee = new(big.Int).Set(parent.blobBaseFee)
	}
	i.traceHook = parent.traceHook
	i.traceDepth = parent.traceDepth + 1
	i.traceDetails = parent.traceDetails
	i.readOnly = parent.readOnly
}

func (i *Interpreter) run(hook func(step TraceStep) bool) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				i.err = err
			} else {
				i.err = fmt.Errorf("execution panic: %v", r)
			}
			i.reverted = true
			i.gas = 0
			logger.Error().Err(i.err).Msg("EVM execution recovered from panic")
		}
	}()

	for i.pc < uint64(len(i.code)) {
		pc := i.pc
		op := i.code[i.pc]
		i.pc++

		if hook != nil {
			pre := i.traceStep(pc, op, false, false)
			if !hook(pre) {
				return
			}
		}

		// Gas deduction
		cost := core.GasTable[op]

		if i.gas < cost {
			i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, cost)
			i.reverted = true
			i.gas = 0
			i.emitPostStep(hook, op, true)
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
		if handler == nil || !i.opcodeEnabled(op) {
			// Log invalid opcode error with context
			logger.Error().
				Uint64("pc", pc).
				Uint8("opcode", op).
				Str("opcode_hex", fmt.Sprintf("0x%02x", op)).
				Int("stack_size", i.stack.Len()).
				Strs("stack", i.stack.Snapshot()).
				Msg("Invalid opcode encountered")

			i.err = fmt.Errorf("unsupported opcode: 0x%02x", op)
			i.reverted = true
			i.gas = 0
			i.emitPostStep(hook, op, true)
			return
		}

		handler(i, op)

		halt := op == core.RETURN || op == core.REVERT || op == core.STOP || i.reverted || i.err != nil
		if i.err != nil {
			i.gas = 0
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

		if !i.emitPostStep(hook, op, halt) || halt {
			return
		}
	}
}

func (i *Interpreter) opcodeEnabled(op byte) bool {
	rules := i.rules()
	switch op {
	case core.DELEGATECALL:
		return rules.IsHomestead
	case core.RETURNDATASIZE, core.RETURNDATACOPY, core.STATICCALL, core.REVERT:
		return rules.IsByzantium
	case core.SHL, core.SHR, core.SAR, core.EXTCODEHASH, core.CREATE2:
		return rules.IsConstantinople
	case core.CHAINID, core.SELFBALANCE:
		return rules.IsIstanbul
	case core.BASEFEE:
		return rules.IsLondon
	case core.PUSH0:
		return rules.IsShanghai
	case core.TLOAD, core.TSTORE, core.MCOPY, core.BLOBHASH, core.BLOBBASEFEE:
		return rules.IsCancun
	case core.CLZ:
		return rules.IsOsaka
	default:
		return true
	}
}

func (i *Interpreter) traceStep(pc uint64, op byte, isPost, halt bool) TraceStep {
	errText := ""
	if i.err != nil {
		errText = i.err.Error()
	}
	step := TraceStep{
		PC:         pc,
		Opcode:     op,
		OpcodeName: core.OpcodeName(op),
		Stack:      i.stack.Snapshot(),
		StackSize:  i.stack.Len(),
		Gas:        i.gas,
		Reverted:   i.reverted,
		Halt:       halt,
		IsPost:     isPost,
		Depth:      i.traceDepth,
		Address:    i.address.Hex(),
		Error:      errText,
	}
	if i.traceDetails {
		step.Memory = append([]byte(nil), i.memory.Data()...)
		step.Storage = i.traceStorageAccesses(op, isPost)
	}
	return step
}

func (i *Interpreter) traceStorageAccesses(op byte, isPost bool) []TraceStorageAccess {
	if isPost {
		return nil
	}
	peekHash := func(index int) (common.Hash, bool) {
		value, err := i.stack.Peek(index)
		if err != nil {
			return common.Hash{}, false
		}
		return common.BigToHash(value), true
	}
	key, ok := peekHash(0)
	if !ok {
		return nil
	}
	access := TraceStorageAccess{
		Address: i.address.Hex(),
		Slot:    key.Hex(),
		Warm:    i.statedb.SlotInAccessList(i.address, key),
	}
	switch op {
	case core.SLOAD:
		access.Kind = "read"
		access.Before = i.statedb.GetState(i.address, key).Hex()
		access.After = access.Before
	case core.SSTORE:
		value, valueOK := peekHash(1)
		if !valueOK {
			return nil
		}
		access.Kind = "write"
		access.Before = i.statedb.GetState(i.address, key).Hex()
		access.After = value.Hex()
		access.Original = i.statedb.GetOriginalState(i.address, key).Hex()
	case core.TLOAD:
		access.Kind = "read"
		access.Transient = true
		access.Before = i.statedb.GetTransientState(i.address, key).Hex()
		access.After = access.Before
	case core.TSTORE:
		value, valueOK := peekHash(1)
		if !valueOK {
			return nil
		}
		access.Kind = "write"
		access.Transient = true
		access.Before = i.statedb.GetTransientState(i.address, key).Hex()
		access.After = value.Hex()
	default:
		return nil
	}
	return []TraceStorageAccess{access}
}

func (i *Interpreter) emitPostStep(hook func(step TraceStep) bool, op byte, halt bool) bool {
	if hook == nil {
		return true
	}
	return hook(i.traceStep(i.pc, op, true, halt))
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
