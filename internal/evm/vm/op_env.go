// op_env.go
package vm

import "math/big"

func opCallValue(i *Interpreter, _ byte) {
	i.stack.Push(big.NewInt(0)) // default to 0
}

// opCallDataSize pushes the size of the calldata onto the stack. Since this
// toy EVM does not support transaction input, it always pushes 0.
func opCallDataSize(i *Interpreter, _ byte) {
	i.stack.Push(big.NewInt(0))
}
