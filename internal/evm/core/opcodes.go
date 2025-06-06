package core

import "fmt"

const (
	// 0x00 - 0x0f: Stop & Arithmetic
	STOP   = 0x00
	ADD    = 0x01
	MUL    = 0x02
	SUB    = 0x03
	DIV    = 0x04
	MOD    = 0x06
	ADDMOD = 0x08

	// 0x10 - 0x1f: Comparison & Bitwise
	LT     = 0x10
	GT     = 0x11
	SLT    = 0x12
	SGT    = 0x13
	EQ     = 0x14
	ISZERO = 0x15
	AND    = 0x16
	OR     = 0x17
	NOT    = 0x19
	SHL    = 0x1b
	SHR    = 0x1c

	// 0x20 - 0x2f: SHA3
	SHA3     = 0x20
	CODECOPY = 0x39

	// 0x30 - 0x4f: Environment
	ADDRESS      = 0x30
	CALLVALUE    = 0x34
	CALLDATALOAD = 0x35
	CALLDATASIZE = 0x36

	// 0x50 - 0x5f: Stack & Memory
	POP      = 0x50
	MLOAD    = 0x51
	MSTORE   = 0x52
	JUMP     = 0x56
	JUMPI    = 0x57
	PC       = 0x58
	JUMPDEST = 0x5b
	PUSH0    = 0x5f // EIP-3855

	// 0x60 - 0x7f: PUSH1 ~ PUSH32
	PUSH1 = 0x60

	// 0x80 - 0x8f: DUP1 ~ DUP16
	DUP1 = 0x80

	// 0x90 - 0x9f: SWAP1 ~ SWAP16
	SWAP1 = 0x90

	// 0xf0+
	RETURN  = 0xf3
	REVERT  = 0xfd
	INVALID = 0xfe
)

var opcodeNames = map[byte]string{
	STOP:         "STOP",
	ADD:          "ADD",
	MUL:          "MUL",
	SUB:          "SUB",
	DIV:          "DIV",
	MOD:          "MOD",
	ADDMOD:       "ADDMOD",
	LT:           "LT",
	GT:           "GT",
	SLT:          "SLT",
	SGT:          "SGT",
	EQ:           "EQ",
	ISZERO:       "ISZERO",
	AND:          "AND",
	OR:           "OR",
	NOT:          "NOT",
	SHL:          "SHL",
	SHR:          "SHR",
	SHA3:         "SHA3",
	ADDRESS:      "ADDRESS",
	CALLVALUE:    "CALLVALUE",
	CALLDATALOAD: "CALLDATALOAD",
	CALLDATASIZE: "CALLDATASIZE",
	POP:          "POP",
	MLOAD:        "MLOAD",
	MSTORE:       "MSTORE",
	JUMP:         "JUMP",
	JUMPI:        "JUMPI",
	PC:           "PC",
	JUMPDEST:     "JUMPDEST",
	PUSH0:        "PUSH0",
	// additional opcodes without constants
	0x05:    "SDIV",
	0x07:    "SMOD",
	0x09:    "MULMOD",
	0x0a:    "EXP",
	0x0b:    "SIGNEXTEND",
	0x18:    "XOR",
	0x1a:    "BYTE",
	0x1d:    "SAR",
	0x1e:    "INVALID",
	0x31:    "BALANCE",
	0x32:    "ORIGIN",
	0x33:    "CALLER",
	0x37:    "CALLDATACOPY",
	0x38:    "CODESIZE",
	0x3a:    "GASPRICE",
	0x3b:    "EXTCODESIZE",
	0x3c:    "EXTCODECOPY",
	0x3d:    "RETURNDATASIZE",
	0x3e:    "RETURNDATACOPY",
	0x3f:    "EXTCODEHASH",
	0x40:    "BLOCKHASH",
	0x41:    "COINBASE",
	0x42:    "TIMESTAMP",
	0x43:    "NUMBER",
	0x44:    "DIFFICULTY",
	0x45:    "GASLIMIT",
	0x46:    "CHAINID",
	0x47:    "SELFBALANCE",
	0x48:    "BASEFEE",
	0x53:    "MSTORE8",
	0x54:    "SLOAD",
	0x55:    "SSTORE",
	0x59:    "MSIZE",
	0x5a:    "GAS",
	0xf0:    "CREATE",
	0xf1:    "CALL",
	0xf2:    "CALLCODE",
	0xf4:    "DELEGATECALL",
	0xf5:    "CREATE2",
	0xfa:    "STATICCALL",
	0xff:    "SELFDESTRUCT",
	RETURN:  "RETURN",
	REVERT:  "REVERT",
	INVALID: "INVALID",
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
