package vm

import (
	"fmt"

	"github.com/smallyunet/echoevm/internal/evm/core"
)

func opMcopy(i *Interpreter, op byte) {
	// Stack: dest, src, length
	dest, err := i.stack.Pop()
	if err != nil {
		i.err = err
		i.reverted = true
		return
	}
	src, err := i.stack.Pop()
	if err != nil {
		i.err = err
		i.reverted = true
		return
	}
	length, err := i.stack.Pop()
	if err != nil {
		i.err = err
		i.reverted = true
		return
	}
	// In a real EVM we handle huge numbers, but for simplicity we can error or cap
	// Standard Go implementation handles uint64 offsets mostly
	// For now assuming uint64 fits
	lenVal := length.Uint64()
	if lenVal == 0 {
		return
	}
	
	// Dynamic gas: 3 * words
	words := (lenVal + 31) / 32
	cost := core.GasCopy * words
	if i.gas < cost {
		i.err = fmt.Errorf("out of gas: mcopy dynamic cost %d", cost)
		i.reverted = true
		return
	}
	i.gas -= cost

	// Memory expansion
	destVal := dest.Uint64()
	srcVal := src.Uint64()
	
	// Max offset is the greater of (dest+len) or (src+len)
	// Actually we need to check expansion for receiving area and reading area?
	// MCOPY reads from memory and writes to memory. So we need to ensure both ranges are within bounds?
	// Actually standard MCOPY only charges expansion for "written" memory usually?
	// Wait, MLOAD expands memory too. So reading also expands.
	// We need to expand to cover max(dest+len, src+len)
	
	maxOffset := destVal
	if srcVal > maxOffset {
		maxOffset = srcVal
	}
	if !i.consumeMemoryExpansion(maxOffset, lenVal) {
		return
	}

	i.memory.Copy(destVal, srcVal, lenVal)
}
