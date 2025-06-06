package core

import "fmt"

const (
	// 0x00 - 0x0f: Stop & Arithmetic
	STOP       = 0x00
	ADD        = 0x01
	MUL        = 0x02
	SUB        = 0x03
	DIV        = 0x04
	SDIV       = 0x05
	MOD        = 0x06
	SMOD       = 0x07
	ADDMOD     = 0x08
	MULMOD     = 0x09
	EXP        = 0x0a
	SIGNEXTEND = 0x0b

	// 0x10 - 0x1f: Comparison & Bitwise
	LT        = 0x10
	GT        = 0x11
	SLT       = 0x12
	SGT       = 0x13
	EQ        = 0x14
	ISZERO    = 0x15
	AND       = 0x16
	OR        = 0x17
	XOR       = 0x18
	NOT       = 0x19
	BYTE      = 0x1a
	SHL       = 0x1b
	SHR       = 0x1c
	SAR       = 0x1d
	INVALID1E = 0x1e

	// 0x20 - 0x2f: SHA3
	SHA3 = 0x20

	// 0x30 - 0x4f: Environment
	ADDRESS        = 0x30
	BALANCE        = 0x31
	ORIGIN         = 0x32
	CALLER         = 0x33
	CALLVALUE      = 0x34
	CALLDATALOAD   = 0x35
	CALLDATASIZE   = 0x36
	CALLDATACOPY   = 0x37
	CODESIZE       = 0x38
	CODECOPY       = 0x39
	GASPRICE       = 0x3a
	EXTCODESIZE    = 0x3b
	EXTCODECOPY    = 0x3c
	RETURNDATASIZE = 0x3d
	RETURNDATACOPY = 0x3e
	EXTCODEHASH    = 0x3f
	BLOCKHASH      = 0x40
	COINBASE       = 0x41
	TIMESTAMP      = 0x42
	NUMBER         = 0x43
	DIFFICULTY     = 0x44
	GASLIMIT       = 0x45
	CHAINID        = 0x46
	SELFBALANCE    = 0x47
	BASEFEE        = 0x48

	// 0x50 - 0x5f: Stack & Memory
	POP      = 0x50
	MLOAD    = 0x51
	MSTORE   = 0x52
	MSTORE8  = 0x53
	SLOAD    = 0x54
	SSTORE   = 0x55
	JUMP     = 0x56
	JUMPI    = 0x57
	PC       = 0x58
	MSIZE    = 0x59
	GAS      = 0x5a
	JUMPDEST = 0x5b
	PUSH0    = 0x5f // EIP-3855

	// 0x60 - 0x7f: PUSH1 ~ PUSH32
	PUSH1 = 0x60

	// 0x80 - 0x8f: DUP1 ~ DUP16
	DUP1 = 0x80

	// 0x90 - 0x9f: SWAP1 ~ SWAP16
	SWAP1 = 0x90

	// 0xf0+
	CREATE       = 0xf0
	CALL         = 0xf1
	CALLCODE     = 0xf2
	RETURN       = 0xf3
	DELEGATECALL = 0xf4
	CREATE2      = 0xf5
	STATICCALL   = 0xfa
	REVERT       = 0xfd
	INVALID      = 0xfe
	SELFDESTRUCT = 0xff
)

var opcodeNames = map[byte]string{
	STOP:           "STOP",
	ADD:            "ADD",
	MUL:            "MUL",
	SUB:            "SUB",
	DIV:            "DIV",
	MOD:            "MOD",
	ADDMOD:         "ADDMOD",
	SDIV:           "SDIV",
	SMOD:           "SMOD",
	MULMOD:         "MULMOD",
	EXP:            "EXP",
	SIGNEXTEND:     "SIGNEXTEND",
	LT:             "LT",
	GT:             "GT",
	SLT:            "SLT",
	SGT:            "SGT",
	EQ:             "EQ",
	ISZERO:         "ISZERO",
	AND:            "AND",
	OR:             "OR",
	XOR:            "XOR",
	NOT:            "NOT",
	BYTE:           "BYTE",
	SHL:            "SHL",
	SHR:            "SHR",
	SAR:            "SAR",
	INVALID1E:      "INVALID",
	SHA3:           "SHA3",
	ADDRESS:        "ADDRESS",
	BALANCE:        "BALANCE",
	ORIGIN:         "ORIGIN",
	CALLER:         "CALLER",
	CALLVALUE:      "CALLVALUE",
	CALLDATALOAD:   "CALLDATALOAD",
	CALLDATASIZE:   "CALLDATASIZE",
	CALLDATACOPY:   "CALLDATACOPY",
	CODESIZE:       "CODESIZE",
	CODECOPY:       "CODECOPY",
	GASPRICE:       "GASPRICE",
	EXTCODESIZE:    "EXTCODESIZE",
	EXTCODECOPY:    "EXTCODECOPY",
	RETURNDATASIZE: "RETURNDATASIZE",
	RETURNDATACOPY: "RETURNDATACOPY",
	EXTCODEHASH:    "EXTCODEHASH",
	BLOCKHASH:      "BLOCKHASH",
	COINBASE:       "COINBASE",
	TIMESTAMP:      "TIMESTAMP",
	NUMBER:         "NUMBER",
	DIFFICULTY:     "DIFFICULTY",
	GASLIMIT:       "GASLIMIT",
	CHAINID:        "CHAINID",
	SELFBALANCE:    "SELFBALANCE",
	BASEFEE:        "BASEFEE",
	POP:            "POP",
	MLOAD:          "MLOAD",
	MSTORE:         "MSTORE",
	MSTORE8:        "MSTORE8",
	SLOAD:          "SLOAD",
	SSTORE:         "SSTORE",
	JUMP:           "JUMP",
	JUMPI:          "JUMPI",
	PC:             "PC",
	MSIZE:          "MSIZE",
	GAS:            "GAS",
	JUMPDEST:       "JUMPDEST",
	PUSH0:          "PUSH0",
	CREATE:         "CREATE",
	CALL:           "CALL",
	CALLCODE:       "CALLCODE",
	DELEGATECALL:   "DELEGATECALL",
	CREATE2:        "CREATE2",
	STATICCALL:     "STATICCALL",
	SELFDESTRUCT:   "SELFDESTRUCT",
	RETURN:         "RETURN",
	REVERT:         "REVERT",
	INVALID:        "INVALID",
}

// OpcodeName returns the mnemonic name of an opcode byte.
func OpcodeName(op byte) string {
	if op >= PUSH1 && op <= 0x7f {
		return fmt.Sprintf("PUSH%d", op-PUSH1+1)
	}
	if op >= DUP1 && op <= 0x8f {
		return fmt.Sprintf("DUP%d", op-DUP1+1)
	}
	if op >= SWAP1 && op <= 0x9f {
		return fmt.Sprintf("SWAP%d", op-SWAP1+1)
	}
	if op >= 0xa0 && op <= 0xa4 {
		return fmt.Sprintf("LOG%d", op-0xa0)
	}
	if name, ok := opcodeNames[op]; ok {
		return name
	}
	return "UNKNOWN"
}
