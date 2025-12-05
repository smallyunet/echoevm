// op_env.go
package vm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func opAddress(i *Interpreter, _ byte) {
	i.stack.PushSafe(i.address.Big())
}

func opBalance(i *Interpreter, _ byte) {
	addrBig := i.stack.PopSafe()
	addr := common.BigToAddress(addrBig)

	// EIP-2929
	var cost uint64
	if i.statedb.AddressInAccessList(addr) {
		cost = 100 // GasWarmStorageRead
	} else {
		cost = 2600 // GasColdAccountAccess
		i.statedb.AddAddressToAccessList(addr)
	}
	
	// Adjust for already paid base cost
	baseCost := core.GasTable[core.BALANCE]
	if cost > baseCost {
		extra := cost - baseCost
		if i.gas < extra {
			i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, extra)
			i.reverted = true
			return
		}
		i.gas -= extra
	} else {
		refund := baseCost - cost
		i.gas += refund
	}

	balance := i.statedb.GetBalance(addr)
	i.stack.PushSafe(new(big.Int).Set(balance))
}

func opOrigin(i *Interpreter, _ byte) {
	i.stack.PushSafe(i.origin.Big())
}

func opCallValue(i *Interpreter, _ byte) {
	if i.callvalue != nil {
		i.stack.PushSafe(new(big.Int).Set(i.callvalue))
	} else {
		i.stack.PushSafe(big.NewInt(0))
	}
}

func opCaller(i *Interpreter, _ byte) {
	i.stack.PushSafe(i.caller.Big())
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

	if !i.consumeMemoryExpansion(memOffset, size) {
		return
	}

	// Dynamic gas: 3 * words
	words := (size + 31) / 32
	copyCost := words * 3
	if i.gas < copyCost {
		i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, copyCost)
		i.reverted = true
		return
	}
	i.gas -= copyCost

	segment := make([]byte, size)
	if dataOffset < uint64(len(i.calldata)) {
		copy(segment, i.calldata[dataOffset:min(dataOffset+size, uint64(len(i.calldata)))])
	}
	i.memory.Write(memOffset, segment)
}

func opCodeSize(i *Interpreter, _ byte) {
	i.stack.PushSafe(big.NewInt(int64(len(i.code))))
}

func opGasPrice(i *Interpreter, _ byte) {
	if i.gasPrice != nil {
		i.stack.PushSafe(new(big.Int).Set(i.gasPrice))
	} else {
		i.stack.PushSafe(big.NewInt(0))
	}
}

func opExtCodeCopy(i *Interpreter, _ byte) {
	addrBig := i.stack.PopSafe()
	addr := common.BigToAddress(addrBig)

	// EIP-2929
	var cost uint64
	if i.statedb.AddressInAccessList(addr) {
		cost = 100 // GasWarmStorageRead
	} else {
		cost = 2600 // GasColdAccountAccess
		i.statedb.AddAddressToAccessList(addr)
	}
	
	// Adjust for already paid base cost
	baseCost := core.GasTable[core.EXTCODECOPY]
	if cost > baseCost {
		extra := cost - baseCost
		if i.gas < extra {
			i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, extra)
			i.reverted = true
			return
		}
		i.gas -= extra
	} else {
		refund := baseCost - cost
		i.gas += refund
	}

	memOffset := i.stack.PopSafe().Uint64()
	codeOffset := i.stack.PopSafe().Uint64()
	length := i.stack.PopSafe().Uint64()

	if !i.consumeMemoryExpansion(memOffset, length) {
		return
	}

	// Dynamic gas: 3 * words
	words := (length + 31) / 32
	copyCost := words * 3
	if i.gas < copyCost {
		i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, copyCost)
		i.reverted = true
		return
	}
	i.gas -= copyCost

	code := i.statedb.GetCode(addr)
	codeCopy := make([]byte, length)
	if codeOffset < uint64(len(code)) {
		copy(codeCopy, code[codeOffset:min(codeOffset+length, uint64(len(code)))])
	}
	i.memory.Write(memOffset, codeCopy)
}

func opReturnDataSize(i *Interpreter, _ byte) {
	i.stack.PushSafe(big.NewInt(int64(len(i.returnData))))
}

func opReturnDataCopy(i *Interpreter, _ byte) {
	memOffset := i.stack.PopSafe().Uint64()
	dataOffset := i.stack.PopSafe().Uint64()
	length := i.stack.PopSafe().Uint64()

	if !i.consumeMemoryExpansion(memOffset, length) {
		return
	}

	// Dynamic gas: 3 * words
	words := (length + 31) / 32
	copyCost := words * 3
	if i.gas < copyCost {
		i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, copyCost)
		i.reverted = true
		return
	}
	i.gas -= copyCost

	data := make([]byte, length)
	if dataOffset < uint64(len(i.returnData)) {
		copy(data, i.returnData[dataOffset:min(dataOffset+length, uint64(len(i.returnData)))])
	}
	i.memory.Write(memOffset, data)
}

func opExtCodeHash(i *Interpreter, _ byte) {
	addrBig := i.stack.PopSafe()
	addr := common.BigToAddress(addrBig)

	// EIP-2929
	var cost uint64
	if i.statedb.AddressInAccessList(addr) {
		cost = 100 // GasWarmStorageRead
	} else {
		cost = 2600 // GasColdAccountAccess
		i.statedb.AddAddressToAccessList(addr)
	}
	
	// Adjust for already paid base cost
	baseCost := core.GasTable[core.EXTCODEHASH]
	if cost > baseCost {
		extra := cost - baseCost
		if i.gas < extra {
			i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, extra)
			i.reverted = true
			return
		}
		i.gas -= extra
	} else {
		refund := baseCost - cost
		i.gas += refund
	}

	if !i.statedb.Exist(addr) {
		i.stack.PushSafe(big.NewInt(0))
		return
	}
	hash := i.statedb.GetCodeHash(addr)
	i.stack.PushSafe(hash.Big())
}

func opBlockHash(i *Interpreter, _ byte) {
	// BlockHash requires access to historical block data which we don't have
	// For now, return 0
	_ = i.stack.PopSafe() // block number
	i.stack.PushSafe(big.NewInt(0))
}

func opDifficulty(i *Interpreter, _ byte) {
	// After The Merge, DIFFICULTY opcode returns PREVRANDAO
	if i.random != nil && i.random.Sign() > 0 {
		i.stack.PushSafe(new(big.Int).Set(i.random))
	} else if i.difficulty != nil {
		i.stack.PushSafe(new(big.Int).Set(i.difficulty))
	} else {
		i.stack.PushSafe(big.NewInt(0))
	}
}

func opChainID(i *Interpreter, _ byte) {
	if i.chainID != nil {
		i.stack.PushSafe(new(big.Int).Set(i.chainID))
	} else {
		i.stack.PushSafe(big.NewInt(1)) // default mainnet
	}
}

func opSelfBalance(i *Interpreter, _ byte) {
	balance := i.statedb.GetBalance(i.address)
	i.stack.PushSafe(new(big.Int).Set(balance))
}

func opBaseFee(i *Interpreter, _ byte) {
	if i.baseFee != nil {
		i.stack.PushSafe(new(big.Int).Set(i.baseFee))
	} else {
		i.stack.PushSafe(big.NewInt(0))
	}
}

func opPC(i *Interpreter, _ byte) {
	// PC points to the current instruction, but pc has already been incremented
	// So we return pc - 1
	i.stack.PushSafe(big.NewInt(int64(i.pc - 1)))
}

func opMSize(i *Interpreter, _ byte) {
	i.stack.PushSafe(big.NewInt(int64(i.memory.Len())))
}

func opGas(i *Interpreter, _ byte) {
	i.stack.PushSafe(new(big.Int).SetUint64(i.gas))
}

func opNumber(i *Interpreter, _ byte) {
	i.stack.PushSafe(big.NewInt(int64(i.blockNumber)))
}

func opTimestamp(i *Interpreter, _ byte) {
	i.stack.PushSafe(big.NewInt(int64(i.timestamp)))
}

func opCoinbase(i *Interpreter, _ byte) {
	i.stack.PushSafe(new(big.Int).SetBytes(i.coinbase.Bytes()))
}

func opGasLimit(i *Interpreter, _ byte) {
	i.stack.PushSafe(big.NewInt(int64(i.gasLimit)))
}

func opExtCodeSize(i *Interpreter, _ byte) {
	addrBig := i.stack.PopSafe()
	addr := common.BigToAddress(addrBig)

	// EIP-2929
	var cost uint64
	if i.statedb.AddressInAccessList(addr) {
		cost = 100 // GasWarmStorageRead
	} else {
		cost = 2600 // GasColdAccountAccess
		i.statedb.AddAddressToAccessList(addr)
	}
	
	// Adjust for already paid base cost
	baseCost := core.GasTable[core.EXTCODESIZE]
	if cost > baseCost {
		extra := cost - baseCost
		if i.gas < extra {
			i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, extra)
			i.reverted = true
			return
		}
		i.gas -= extra
	} else {
		refund := baseCost - cost
		i.gas += refund
	}

	size := i.statedb.GetCodeSize(addr)
	i.stack.PushSafe(big.NewInt(int64(size)))
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
