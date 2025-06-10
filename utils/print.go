// utils.go or in main.go for now
package utils

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

// PrintBytecode prints EVM bytecode in a human-readable format.
// For each opcode, it shows the program counter (PC), opcode hex value,
// and the mnemonic name (e.g., ADD, PUSH1), with push data if applicable.
// PrintBytecode logs EVM bytecode in a human-readable format using the provided logger.
func PrintBytecode(log zerolog.Logger, code []byte) {
	log.Info().Msg("=== Bytecode ===")

	// pc represents the current position in the bytecode (like the program counter in EVM)
	for pc := 0; pc < len(code); {
		op := code[pc]
		line := fmt.Sprintf("0x%04x:  0x%02x", pc, op)

		// Handle PUSH1 to PUSH32 opcodes (0x60 to 0x7f), which include N bytes of immediate data
		if op >= 0x60 && op <= 0x7f {
			// Calculate number of bytes to push (PUSH1 pushes 1 byte, PUSH32 pushes 32 bytes, etc.)
			n := int(op - 0x60 + 1)

			// Ensure there are enough bytes left in code for the push data
			if pc+n >= len(code) {
				line += "  [invalid push: out of bounds]"
			} else {
				line += fmt.Sprintf("  PUSH%d 0x", n)
				for i := 0; i < n; i++ {
					line += fmt.Sprintf("%02x", code[pc+1+i])
				}
			}

			// Move program counter forward by opcode byte + N bytes of data
			pc += n + 1
		} else {
			// For non-PUSH opcodes, print their mnemonic name
			line += fmt.Sprintf("  %s", opcodeName(op))
			// Move to next byte (each non-PUSH opcode is 1 byte)
			pc += 1
		}
		log.Info().Msg(line)
	}
}

// opcodeName returns the mnemonic name of an EVM opcode byte for readability.
func opcodeName(op byte) string {
	return core.OpcodeName(op)
}
