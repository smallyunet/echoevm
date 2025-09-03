// op_control.go
package vm

import (
	"fmt"
)

func opStop(_ *Interpreter, _ byte) {
	// halt execution
}

func opReturn(i *Interpreter, _ byte) {
	offset := i.stack.PopSafe().Uint64()
	size := i.stack.PopSafe().Uint64()
	ret := i.memory.Read(offset, size)
	i.returned = ret

	// Log RETURN operation with structured data
	logger.Info().
		Uint64("offset", offset).
		Uint64("size", size).
		Str("return_data_hex", fmt.Sprintf("0x%x", ret)).
		Int("return_data_size", len(ret)).
		Msg("RETURN operation executed")
}

// opRevert halts execution and marks the returned data as an error payload.
// This simplified EVM does not differentiate between revert and return beyond
// printing a message and storing the payload.
func opRevert(i *Interpreter, _ byte) {
	offset := i.stack.PopSafe().Uint64()
	size := i.stack.PopSafe().Uint64()
	ret := i.memory.Read(offset, size)
	i.returned = ret
	i.reverted = true

	// Log REVERT operation with structured data
	logger.Error().
		Uint64("offset", offset).
		Uint64("size", size).
		Str("revert_data_hex", fmt.Sprintf("0x%x", ret)).
		Int("revert_data_size", len(ret)).
		Msg("REVERT operation executed")
}
