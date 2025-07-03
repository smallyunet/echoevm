package vm

import "math/big"

// opDelegateCall is a stub that always fails. It pops the expected
// arguments and pushes 0 to indicate failure.
func opDelegateCall(i *Interpreter, _ byte) {
	for n := 0; n < 6; n++ {
		if i.stack.Len() > 0 {
			i.stack.Pop()
		}
	}
	i.stack.Push(big.NewInt(0))
}
