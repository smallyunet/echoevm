package vm

func opInvalid(i *Interpreter, op byte) {
	// In a proper EVM implementation, this should cause a revert
	// For now, we'll just set the reverted flag
	i.reverted = true
	// Note: In a real implementation, you might want to return an error instead
	// of just setting the reverted flag
}
