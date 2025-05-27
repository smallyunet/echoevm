// op_env.go
package vm

import "math/big"

func opCallValue(i *Interpreter, _ byte) {
	i.stack.Push(big.NewInt(0)) // default to 0
}
