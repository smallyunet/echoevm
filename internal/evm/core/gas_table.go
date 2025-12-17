package core

// Gas costs based on EIP-2929 (Berlin) and later specifications
const (
	GasZero      = 0
	GasBase      = 2
	GasVeryLow   = 3
	GasLow       = 5
	GasMid       = 8
	GasHigh      = 10
	GasExtCode   = 700
	GasBalance   = 400  // Warm access (after access list)
	GasSload     = 100  // Warm storage read (EIP-2929)
	GasJumpDest  = 1
	GasSstoreSet = 20000
	GasSstoreReset = 2900
	GasSstoreClear = 4800 // Refund for clearing storage
	GasLog       = 375
	GasLogData   = 8
	GasLogTopic  = 375
	GasCreate    = 32000
	GasCall      = 100   // Warm call (EIP-2929)
	GasCallStipend = 2300
	GasCallValue = 9000
	GasCallNewAccount = 25000
	GasSelfDestruct = 5000
	GasSelfDestructNewAccount = 25000
	GasKeccak256 = 30
	GasKeccak256Word = 6
	GasCopy = 3
	GasBlockhash = 20
	GasExpByte   = 50   // Per byte of exponent
	
	// EIP-2929: Access list costs
	GasWarmStorageRead = 100
	GasColdStorageRead = 2100
	GasColdAccountAccess = 2600
	GasColdSload = 2100

	// EIP-3529: Reduced refunds
	GasSstoreClearRefund = 4800

	// EIP-1153: Transient Storage
	GasTload  = 100
	GasTstore = 100

	// EIP-5656: MCOPY
	GasMcopy = 3

	// Memory costs
	GasMemory = 3 // Per word

	// Transaction costs
	GasTxDataZero    = 4
	GasTxDataNonZero = 16
	GasTxCreate      = 32000
	GasTxBase        = 21000
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
	TLOAD:    GasTload,
	TSTORE:   GasTstore,
	MCOPY:    GasMcopy, // Dynamic cost added in handler

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

// ExpGasCost calculates gas for EXP opcode: 10 + 50 * byte_len(exponent)
func ExpGasCost(exponent []byte) uint64 {
	// Count significant bytes (skip leading zeros)
	significantBytes := uint64(0)
	for _, b := range exponent {
		if b != 0 {
			significantBytes = uint64(len(exponent)) - uint64(0) // All bytes from first non-zero
			break
		}
	}
	if significantBytes == 0 {
		return GasHigh // exp(x, 0) = 1, base cost only
	}

	byteLen := uint64(0)
	for i, b := range exponent {
		if b != 0 {
			byteLen = uint64(len(exponent) - i)
			break
		}
	}
	return GasHigh + GasExpByte*byteLen
}

// Sha3GasCost calculates gas for SHA3 opcode: 30 + 6 * word_count
func Sha3GasCost(size uint64) uint64 {
	words := (size + 31) / 32
	return GasKeccak256 + GasKeccak256Word*words
}

// CopyGasCost calculates gas for CALLDATACOPY, CODECOPY, etc.: 3 * word_count
func CopyGasCost(size uint64) uint64 {
	words := (size + 31) / 32
	return GasCopy * words
}

// LogGasCost calculates gas for LOG opcodes: 375 + 375*topics + 8*data_size
func LogGasCost(topics int, dataSize uint64) uint64 {
	return GasLog + uint64(topics)*GasLogTopic + dataSize*GasLogData
}

// SstoreGasCost calculates gas for SSTORE based on EIP-2200/EIP-3529
// Returns (gasCost, refund)
func SstoreGasCost(original, current, newVal [32]byte, isWarm bool) (uint64, uint64) {
	var baseCost uint64
	if !isWarm {
		baseCost = GasColdStorageRead
	}

	// No-op: value unchanged
	if current == newVal {
		return baseCost + GasWarmStorageRead, 0
	}

	// Fresh slot (original is zero)
	if original == [32]byte{} {
		if newVal == [32]byte{} {
			return baseCost + GasWarmStorageRead, 0
		}
		return baseCost + GasSstoreSet, 0
	}

	// Existing slot modification
	if current == original {
		if newVal == [32]byte{} {
			// Clearing storage
			return baseCost + GasSstoreReset, GasSstoreClearRefund
		}
		return baseCost + GasSstoreReset, 0
	}

	// Dirty slot
	var refund uint64
	if original != [32]byte{} {
		if current == [32]byte{} && newVal != [32]byte{} {
			// Un-clearing
			refund = 0 // No longer give refund for recreating
		} else if current != [32]byte{} && newVal == [32]byte{} {
			refund = GasSstoreClearRefund
		}
	}
	if newVal == original {
		// Restoring to original
		if original == [32]byte{} {
			refund += GasSstoreSet - GasWarmStorageRead
		} else {
			refund += GasSstoreReset - GasWarmStorageRead
		}
	}

	return baseCost + GasWarmStorageRead, refund
}

// CallGasCost calculates gas for CALL-family opcodes
func CallGasCost(isWarm bool, hasValue bool, isNewAccount bool, gas uint64) uint64 {
	var cost uint64
	
	if !isWarm {
		cost = GasColdAccountAccess
	} else {
		cost = GasCall
	}

	if hasValue {
		cost += GasCallValue
		if isNewAccount {
			cost += GasCallNewAccount
		}
	}

	return cost
}

// TxDataGasCost calculates gas for transaction data
func TxDataGasCost(data []byte) uint64 {
	var cost uint64
	for _, b := range data {
		if b == 0 {
			cost += GasTxDataZero
		} else {
			cost += GasTxDataNonZero
		}
	}
	return cost
}

// IntrinsicGas calculates the intrinsic gas for a transaction
func IntrinsicGas(data []byte, isCreate bool) uint64 {
	gas := uint64(GasTxBase)
	if isCreate {
		gas += GasTxCreate
	}
	gas += TxDataGasCost(data)
	return gas
}
