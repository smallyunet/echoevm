package core

// Gas costs
const (
	GasZero      = 0
	GasBase      = 2
	GasVeryLow   = 3
	GasLow       = 5
	GasMid       = 8
	GasHigh      = 10
	GasExtCode   = 700
	GasBalance   = 400
	GasSload     = 800
	GasJumpDest  = 1
	GasSstoreSet = 20000
	GasSstoreReset = 2900
	GasSstoreClear = 2900 // Simplified
	GasLog       = 375
	GasLogData   = 8
	GasLogTopic  = 375
	GasCreate    = 32000
	GasCall      = 700
	GasCallStipend = 2300
	GasCallValue = 9000
	GasCallNewAccount = 25000
	GasSelfDestruct = 5000
	GasKeccak256 = 30
	GasKeccak256Word = 6
	GasCopy = 3
	GasBlockhash = 20
	
	// EIP-2929: Access list costs
	GasWarmStorageRead = 100
	GasColdAccountAccess = 2600
)

// GasTable maps opcodes to their base gas cost
var GasTable = [256]uint64{
	STOP:       GasZero,
	ADD:        GasVeryLow,
	MUL:        GasLow,
	SUB:        GasVeryLow,
	DIV:        GasLow,
	SDIV:       GasLow,
	MOD:        GasLow,
	SMOD:       GasLow,
	ADDMOD:     GasMid,
	MULMOD:     GasMid,
	EXP:        GasHigh, // Dynamic cost not included here
	SIGNEXTEND: GasLow,

	LT:     GasVeryLow,
	GT:     GasVeryLow,
	SLT:    GasVeryLow,
	SGT:    GasVeryLow,
	EQ:     GasVeryLow,
	ISZERO: GasVeryLow,
	AND:    GasVeryLow,
	OR:     GasVeryLow,
	XOR:    GasVeryLow,
	NOT:    GasVeryLow,
	BYTE:   GasVeryLow,
	SHL:    GasVeryLow,
	SHR:    GasVeryLow,
	SAR:    GasVeryLow,

	SHA3: GasKeccak256, // Dynamic cost not included

	ADDRESS:        GasBase,
	BALANCE:        GasBalance,
	ORIGIN:         GasBase,
	CALLER:         GasBase,
	CALLVALUE:      GasBase,
	CALLDATALOAD:   GasVeryLow,
	CALLDATASIZE:   GasBase,
	CALLDATACOPY:   GasVeryLow, // Dynamic
	CODESIZE:       GasBase,
	CODECOPY:       GasVeryLow, // Dynamic
	GASPRICE:       GasBase,
	EXTCODESIZE:    GasExtCode,
	EXTCODECOPY:    GasExtCode, // Dynamic
	RETURNDATASIZE: GasBase,
	RETURNDATACOPY: GasVeryLow, // Dynamic
	EXTCODEHASH:    GasExtCode,
	BLOCKHASH:      GasBlockhash,
	COINBASE:       GasBase,
	TIMESTAMP:      GasBase,
	NUMBER:         GasBase,
	DIFFICULTY:     GasBase,
	GASLIMIT:       GasBase,
	CHAINID:        GasBase,
	SELFBALANCE:    GasLow,
	BASEFEE:        GasBase,

	POP:      GasBase,
	MLOAD:    GasVeryLow,
	MSTORE:   GasVeryLow,
	MSTORE8:  GasVeryLow,
	SLOAD:    GasSload,
	SSTORE:   GasZero, // Dynamic
	JUMP:     GasMid,
	JUMPI:    GasHigh,
	PC:       GasBase,
	MSIZE:    GasBase,
	GAS:      GasBase,
	JUMPDEST: GasJumpDest,
	PUSH0:    GasBase,

	LOG0: GasLog,
	LOG1: GasLog + GasLogTopic,
	LOG2: GasLog + 2*GasLogTopic,
	LOG3: GasLog + 3*GasLogTopic,
	LOG4: GasLog + 4*GasLogTopic,

	CREATE:       GasCreate,
	CALL:         GasCall,
	CALLCODE:     GasCall,
	RETURN:       GasZero,
	DELEGATECALL: GasCall,
	CREATE2:      GasCreate,
	STATICCALL:   GasCall,
	REVERT:       GasZero,
	INVALID:      GasZero,
	SELFDESTRUCT: GasSelfDestruct,
}

func init() {
	// Fill PUSH, DUP, SWAP
	for i := 0; i < 32; i++ {
		GasTable[PUSH1+byte(i)] = GasVeryLow
	}
	for i := 0; i < 16; i++ {
		GasTable[DUP1+byte(i)] = GasVeryLow
		GasTable[SWAP1+byte(i)] = GasVeryLow
	}
}

func MemoryGasCost(size uint64) uint64 {
	words := (size + 31) / 32
	return (words*words)/512 + (3 * words)
}
