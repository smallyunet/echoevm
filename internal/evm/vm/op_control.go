// op_control.go
package vm

func opStop(_ *Interpreter, _ byte) {
	// halt execution
}

func opReturn(i *Interpreter, _ byte) {
	offset := i.stack.Pop().Uint64()
	size := i.stack.Pop().Uint64()
	ret := i.memory.Read(offset, size)
	i.returned = ret
	logger.Info().Msgf("RETURN: 0x%x", ret)
}

// opRevert halts execution and marks the returned data as an error payload.
// This simplified EVM does not differentiate between revert and return beyond
// printing a message and storing the payload.
func opRevert(i *Interpreter, _ byte) {
	offset := i.stack.Pop().Uint64()
	size := i.stack.Pop().Uint64()
	ret := i.memory.Read(offset, size)
	i.returned = ret
	logger.Info().Msgf("REVERT: 0x%x", ret)
}
