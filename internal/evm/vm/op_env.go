// op_env.go
package vm

import "math/big"

func opCallValue(i *Interpreter, _ byte) {
	i.stack.PushSafe(big.NewInt(0)) // default to 0
}

// opCaller pushes the address of the caller. Since this interpreter does not
// model accounts, the value is always zero.
func opCaller(i *Interpreter, _ byte) {
	i.stack.PushSafe(big.NewInt(0))
}

// opCallDataSize pushes the size of the calldata onto the stack. If no calldata
// is provided it returns 0.
func opCallDataSize(i *Interpreter, _ byte) {
	i.stack.PushSafe(big.NewInt(int64(len(i.calldata))))
}

// opCallDataLoad pushes 32 bytes from calldata starting at the given offset
// onto the stack. If the requested bytes exceed the calldata length, the
// missing bytes are treated as zero.
func opCallDataLoad(i *Interpreter, _ byte) {
	offset := i.stack.PopSafe().Uint64()
	end := offset + 32
	data := make([]byte, 32)
	if offset < uint64(len(i.calldata)) {
		copy(data, i.calldata[offset:min(end, uint64(len(i.calldata)))])
	}
	i.stack.PushSafe(new(big.Int).SetBytes(data))
}

// opCallDataCopy copies a slice of calldata into memory. The stack provides the
// destination memory offset, the calldata offset and the size to copy.
func opCallDataCopy(i *Interpreter, _ byte) {
	memOffset := i.stack.PopSafe().Uint64()
	dataOffset := i.stack.PopSafe().Uint64()
	size := i.stack.PopSafe().Uint64()
	segment := make([]byte, size)
	if dataOffset < uint64(len(i.calldata)) {
		copy(segment, i.calldata[dataOffset:min(dataOffset+size, uint64(len(i.calldata)))])
	}
	i.memory.Write(memOffset, segment)
}

func opGas(i *Interpreter, _ byte) {
	i.stack.PushSafe(big.NewInt(0))
}

func opNumber(i *Interpreter, _ byte) {
	i.stack.PushSafe(big.NewInt(int64(i.blockNumber)))
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
