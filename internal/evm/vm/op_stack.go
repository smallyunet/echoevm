// op_stack.go
package vm

import (
	"fmt"
	"math/big"
)

func opPush0(i *Interpreter, _ byte) {
	i.stack.PushSafe(big.NewInt(0))
}

func opPop(i *Interpreter, _ byte) {
	i.stack.PopSafe()
}

func opPush(i *Interpreter, op byte) {
	n := int(op - 0x5f)
	if int(i.pc)+n > len(i.code) {
		// Instead of panicking, we'll set the reverted flag
		i.err = fmt.Errorf("push out of bounds: pc=%d, n=%d, len=%d", i.pc, n, len(i.code))
		i.reverted = true
		return
	}
	val := new(big.Int).SetBytes(i.code[i.pc : i.pc+uint64(n)])
	i.pc += uint64(n)
	i.stack.PushSafe(val)
}

func opDup(i *Interpreter, op byte) {
	n := int(op - 0x7f)
	i.stack.DupSafe(n)
}

func opSwap(i *Interpreter, op byte) {
	n := int(op - 0x8f)
	i.stack.SwapSafe(n)
}
