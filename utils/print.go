// utils.go or in main.go for now
package utils

import (
	"fmt"
)

// PrintBytecode prints EVM bytecode in a human-readable format.
// For each opcode, it shows the program counter (PC), opcode hex value,
// and the mnemonic name (e.g., ADD, PUSH1), with push data if applicable.
func PrintBytecode(code []byte) {
	fmt.Println("=== Bytecode ===")

	// pc represents the current position in the bytecode (like the program counter in EVM)
	for pc := 0; pc < len(code); {
		op := code[pc]
		fmt.Printf("0x%04x:  0x%02x", pc, op)

		// Handle PUSH1 to PUSH32 opcodes (0x60 to 0x7f), which include N bytes of immediate data
		if op >= 0x60 && op <= 0x7f {
			// Calculate number of bytes to push (PUSH1 pushes 1 byte, PUSH32 pushes 32 bytes, etc.)
			n := int(op - 0x60 + 1)

			// Ensure there are enough bytes left in code for the push data
			if pc+n >= len(code) {
				fmt.Print("  [invalid push: out of bounds]")
			} else {
				fmt.Printf("  PUSH%d 0x", n)
				// Print each byte of the immediate push data in hex
				for i := 0; i < n; i++ {
					fmt.Printf("%02x", code[pc+1+i])
				}
			}

			// Move program counter forward by opcode byte + N bytes of data
			pc += n + 1
		} else {
			// For non-PUSH opcodes, print their mnemonic name
			fmt.Print("  ", opcodeName(op))
			// Move to next byte (each non-PUSH opcode is 1 byte)
			pc += 1
		}
		fmt.Println()
	}
}

// opcodeName returns the mnemonic name of an EVM opcode byte for readability.
func opcodeName(op byte) string {
	// Handle PUSH1 ~ PUSH32
	if op >= 0x60 && op <= 0x7f {
		return fmt.Sprintf("PUSH%d", op-0x5f)
	}
	// Handle DUP1 ~ DUP16
	if op >= 0x80 && op <= 0x8f {
		return fmt.Sprintf("DUP%d", op-0x7f)
	}
	// Handle SWAP1 ~ SWAP16
	if op >= 0x90 && op <= 0x9f {
		return fmt.Sprintf("SWAP%d", op-0x8f)
	}
	// Handle LOG0 ~ LOG4
	if op >= 0xa0 && op <= 0xa4 {
		return fmt.Sprintf("LOG%d", op-0xa0)
	}

	names := map[byte]string{
		0x00: "STOP",
		0x01: "ADD",
		0x02: "MUL",
		0x03: "SUB",
		0x04: "DIV",
		0x05: "SDIV",
		0x06: "MOD",
		0x07: "SMOD",
		0x08: "ADDMOD",
		0x09: "MULMOD",
		0x0a: "EXP",
		0x0b: "SIGNEXTEND",

		0x10: "LT",
		0x11: "GT",
		0x12: "SLT",
		0x13: "SGT",
		0x14: "EQ",
		0x15: "ISZERO",
		0x16: "AND",
		0x17: "OR",
		0x18: "XOR",
		0x19: "NOT",
		0x1a: "BYTE",
		0x1b: "SHL",
		0x1c: "SHR",
		0x1d: "SAR",
		0x1e: "INVALID", // 你 bytecode 中的 0x1e 实际是未知/废弃，先保留为 INVALID

		0x20: "SHA3",

		0x30: "ADDRESS",
		0x31: "BALANCE",
		0x32: "ORIGIN",
		0x33: "CALLER",
		0x34: "CALLVALUE",
		0x35: "CALLDATALOAD",
		0x36: "CALLDATASIZE",
		0x37: "CALLDATACOPY",
		0x38: "CODESIZE",
		0x39: "CODECOPY",
		0x3a: "GASPRICE",
		0x3b: "EXTCODESIZE",
		0x3c: "EXTCODECOPY",
		0x3d: "RETURNDATASIZE",
		0x3e: "RETURNDATACOPY",
		0x3f: "EXTCODEHASH",

		0x40: "BLOCKHASH",
		0x41: "COINBASE",
		0x42: "TIMESTAMP",
		0x43: "NUMBER",
		0x44: "DIFFICULTY",
		0x45: "GASLIMIT",
		0x46: "CHAINID",
		0x47: "SELFBALANCE",
		0x48: "BASEFEE",

		0x50: "POP",
		0x51: "MLOAD",
		0x52: "MSTORE",
		0x53: "MSTORE8",
		0x54: "SLOAD",
		0x55: "SSTORE",
		0x56: "JUMP",
		0x57: "JUMPI",
		0x58: "PC",
		0x59: "MSIZE",
		0x5a: "GAS",
		0x5b: "JUMPDEST",
		0x5f: "PUSH0", // EIP-3855

		0xf0: "CREATE",
		0xf1: "CALL",
		0xf2: "CALLCODE",
		0xf3: "RETURN",
		0xf4: "DELEGATECALL",
		0xf5: "CREATE2",
		0xfa: "STATICCALL",
		0xfd: "REVERT",
		0xfe: "INVALID",
		0xff: "SELFDESTRUCT",
	}

	if name, ok := names[op]; ok {
		return name
	}
	return "UNKNOWN"
}
