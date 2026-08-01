package vm

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

const (
	maxRuntimeCodeSize = 24_576
	maxInitCodeSize    = 2 * maxRuntimeCodeSize
	createDataGas      = 200
	initCodeWordGas    = 2
	keccak256WordGas   = 6
)

// opCreate implements the CREATE opcode.
func opCreate(i *Interpreter, _ byte) {
	if i.rejectWriteProtection() {
		return
	}
	// Stack: value, offset, length
	value := i.stack.PopSafe()
	offset := i.stack.PopSafe().Uint64()
	length := i.stack.PopSafe().Uint64()

	if !i.consumeMemoryExpansion(offset, length) {
		return
	}
	if !i.chargeCreateWordGas(length, initCodeWordGas) {
		return
	}
	if i.statedb.GetBalance(i.address).Cmp(value) < 0 {
		i.stack.PushSafe(big.NewInt(0))
		i.returnData = nil
		return
	}

	initCode := i.memory.Read(offset, length)
	nonce := i.statedb.GetNonce(i.address)
	addr := crypto.CreateAddress(i.address, nonce)
	i.createContract(addr, value, initCode, nonce)
}

// opCreate2 implements the CREATE2 opcode (EIP-1014)
func opCreate2(i *Interpreter, _ byte) {
	if i.rejectWriteProtection() {
		return
	}
	// Stack: value, offset, length, salt
	value := i.stack.PopSafe()
	offset := i.stack.PopSafe().Uint64()
	length := i.stack.PopSafe().Uint64()
	salt := i.stack.PopSafe()

	if !i.consumeMemoryExpansion(offset, length) {
		return
	}
	if !i.chargeCreateWordGas(length, initCodeWordGas+keccak256WordGas) {
		return
	}
	if i.statedb.GetBalance(i.address).Cmp(value) < 0 {
		i.stack.PushSafe(big.NewInt(0))
		i.returnData = nil
		return
	}

	// 2. Get init code
	initCode := i.memory.Read(offset, length)

	// 3. Calculate address: keccak256(0xff ++ sender ++ salt ++ keccak256(initCode))[12:]
	saltBytes := make([]byte, 32)
	salt.FillBytes(saltBytes)

	codeHash := crypto.Keccak256(initCode)

	data := make([]byte, 1+20+32+32)
	data[0] = 0xff
	copy(data[1:21], i.address.Bytes())
	copy(data[21:53], saltBytes)
	copy(data[53:85], codeHash)

	addr := common.BytesToAddress(crypto.Keccak256(data)[12:])

	nonce := i.statedb.GetNonce(i.address)
	i.createContract(addr, value, initCode, nonce)
}

func (i *Interpreter) chargeCreateWordGas(length, perWord uint64) bool {
	if length > maxInitCodeSize {
		i.err = fmt.Errorf("max initcode size exceeded: code size %d limit %d", length, maxInitCodeSize)
		i.reverted = true
		return false
	}
	words := (length + 31) / 32
	cost := words * perWord
	if i.gas < cost {
		i.err = fmt.Errorf("out of gas: initcode word cost")
		i.reverted = true
		return false
	}
	i.gas -= cost
	return true
}

func (i *Interpreter) createContract(addr common.Address, value *big.Int, initCode []byte, creatorNonce uint64) {
	if creatorNonce == math.MaxUint64 {
		i.stack.PushSafe(big.NewInt(0))
		i.returnData = nil
		return
	}
	i.statedb.SetNonce(i.address, creatorNonce+1)
	i.statedb.AddAddressToAccessList(addr)

	available := i.gas
	forwarded := available - available/64
	i.gas -= forwarded

	if i.statedb.GetNonce(addr) != 0 || i.statedb.GetCodeSize(addr) != 0 {
		i.stack.PushSafe(big.NewInt(0))
		i.returnData = nil
		return
	}

	snapshot := i.statedb.Snapshot()
	i.statedb.CreateAccount(addr)
	i.statedb.SetNonce(addr, 1)
	i.statedb.SubBalance(i.address, value)
	i.statedb.AddBalance(addr, value)

	contract := New(initCode, i.statedb, addr)
	contract.inheritExecutionContext(i)
	contract.SetGas(forwarded)
	contract.SetCaller(i.address)
	contract.SetOrigin(i.origin)
	contract.SetCallValue(value)
	contract.Run()

	ret := contract.ReturnedCode()
	if contract.Err() != nil {
		i.statedb.RevertToSnapshot(snapshot)
		i.stack.PushSafe(big.NewInt(0))
		i.returnData = nil
		return
	}
	if contract.IsReverted() {
		i.statedb.RevertToSnapshot(snapshot)
		i.gas += contract.Gas()
		i.stack.PushSafe(big.NewInt(0))
		i.returnData = ret
		return
	}
	if len(ret) > maxRuntimeCodeSize || (len(ret) > 0 && ret[0] == 0xef) {
		i.statedb.RevertToSnapshot(snapshot)
		i.stack.PushSafe(big.NewInt(0))
		i.returnData = nil
		return
	}
	depositCost := uint64(len(ret)) * createDataGas
	if contract.Gas() < depositCost {
		i.statedb.RevertToSnapshot(snapshot)
		i.stack.PushSafe(big.NewInt(0))
		i.returnData = nil
		return
	}
	contract.gas -= depositCost
	i.statedb.SetCode(addr, ret)
	i.gas += contract.Gas()
	i.stack.PushSafe(addr.Big())
	i.returnData = nil
}

// opCall implements the CALL opcode.
func opCall(i *Interpreter, _ byte) {
	// Stack: gas, addr, value, argsOffset, argsLength, retOffset, retLength
	gas := i.stack.PopSafe()
	addrBig := i.stack.PopSafe()
	addr := common.BigToAddress(addrBig)
	value := i.stack.PopSafe()
	argsOffset := i.stack.PopSafe().Uint64()
	argsLength := i.stack.PopSafe().Uint64()
	retOffset := i.stack.PopSafe().Uint64()
	retLength := i.stack.PopSafe().Uint64()
	if i.readOnly && value.Sign() != 0 {
		i.err = ErrWriteProtection
		i.reverted = true
		return
	}

	// Dynamic gas
	var callCost uint64

	// EIP-2929
	var accessCost uint64
	if i.statedb.AddressInAccessList(addr) {
		accessCost = core.GasWarmStorageRead
	} else {
		accessCost = core.GasColdAccountAccess
		i.statedb.AddAddressToAccessList(addr)
	}

	// Adjust for already paid base cost
	baseCost := core.GasTable[core.CALL]
	if accessCost > baseCost {
		callCost += (accessCost - baseCost)
	} else {
		i.gas += (baseCost - accessCost)
	}

	if value.Sign() > 0 {
		callCost += 9000 // GasCallValue
		if !i.statedb.Exist(addr) {
			callCost += 25000 // GasCallNewAccount
		}
	}
	if i.gas < callCost {
		i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, callCost)
		i.reverted = true
		return
	}
	i.gas -= callCost

	// 1. Snapshot state before call
	snapshot := i.statedb.Snapshot()

	// 2. Transfer value
	if value.Sign() > 0 && i.statedb.GetBalance(i.address).Cmp(value) < 0 {
		i.stack.PushSafe(big.NewInt(0))
		i.returnData = nil
		return
	}
	if value.Sign() > 0 {
		i.statedb.SubBalance(i.address, value)
		i.statedb.AddBalance(addr, value)
	}

	// 3. Get code
	code := i.statedb.GetCode(addr)

	if !i.consumeMemoryExpansion(argsOffset, argsLength) {
		i.statedb.RevertToSnapshot(snapshot)
		return
	}
	if !i.consumeMemoryExpansion(retOffset, retLength) {
		i.statedb.RevertToSnapshot(snapshot)
		return
	}

	// 4. Get input data
	args := i.memory.Read(argsOffset, argsLength)

	// Handle gas passing (EIP-150)
	gasLimit := gas.Uint64()
	available := i.gas
	cap := available - available/64
	if gasLimit > cap {
		gasLimit = cap
	}
	i.gas -= gasLimit

	// Add call stipend if value is transferred
	if value.Sign() > 0 {
		gasLimit += core.GasCallStipend
	}

	// 5. Check for precompiled contracts
	if IsPrecompiled(addr) {
		ret, remainingGas, err := RunPrecompiled(addr, args, gasLimit)
		i.returnData = ret
		i.gas += remainingGas

		if err != nil {
			i.statedb.RevertToSnapshot(snapshot)
			i.stack.PushSafe(big.NewInt(0))
		} else {
			// Copy return data to memory
			toCopy := uint64(len(ret))
			if toCopy > retLength {
				toCopy = retLength
			}
			if toCopy > 0 {
				i.memory.Write(retOffset, ret[:toCopy])
			}
			i.stack.PushSafe(big.NewInt(1))
		}
		return
	}

	// 6. Execute regular contract
	contract := NewWithCallData(code, args, i.statedb, addr)
	contract.inheritExecutionContext(i)
	contract.SetBlockNumber(i.blockNumber)
	contract.SetTimestamp(i.timestamp)
	contract.SetCoinbase(i.coinbase)
	contract.SetBlockGasLimit(i.gasLimit)
	contract.SetGas(gasLimit)

	contract.SetCaller(i.address)
	contract.SetOrigin(i.origin)
	contract.SetCallValue(value)
	contract.SetChainID(i.chainID)
	contract.SetGasPrice(i.gasPrice)

	contract.Run()

	// 6. Store return data
	ret := contract.ReturnedCode()
	i.returnData = ret

	// 7. Handle errors and revert
	if contract.Err() != nil {
		// Error (not clean revert): consume all gas
		i.statedb.RevertToSnapshot(snapshot)
		i.stack.PushSafe(big.NewInt(0))
		// Copy return data even on failure
		toCopy := uint64(len(ret))
		if toCopy > retLength {
			toCopy = retLength
		}
		if toCopy > 0 {
			i.memory.Write(retOffset, ret[:toCopy])
		}
	} else if contract.IsReverted() {
		// Clean revert (REVERT opcode): return remaining gas
		i.gas += contract.Gas()
		i.statedb.RevertToSnapshot(snapshot)
		i.stack.PushSafe(big.NewInt(0))
		// Copy return data even on failure
		toCopy := uint64(len(ret))
		if toCopy > retLength {
			toCopy = retLength
		}
		if toCopy > 0 {
			i.memory.Write(retOffset, ret[:toCopy])
		}
	} else {
		// Success: return remaining gas
		i.gas += contract.Gas()
		// 8. Copy to memory on success
		toCopy := uint64(len(ret))
		if toCopy > retLength {
			toCopy = retLength
		}
		if toCopy > 0 {
			i.memory.Write(retOffset, ret[:toCopy])
		}
		i.stack.PushSafe(big.NewInt(1))
	}
}

// opCallCode implements the CALLCODE opcode.
// Similar to CALL but executes code in the context of the caller.
func opCallCode(i *Interpreter, _ byte) {
	// Stack: gas, addr, value, argsOffset, argsLength, retOffset, retLength
	gas := i.stack.PopSafe()
	addrBig := i.stack.PopSafe()
	addr := common.BigToAddress(addrBig)
	value := i.stack.PopSafe()
	argsOffset := i.stack.PopSafe().Uint64()
	argsLength := i.stack.PopSafe().Uint64()
	retOffset := i.stack.PopSafe().Uint64()
	retLength := i.stack.PopSafe().Uint64()

	// Dynamic gas
	var callCost uint64

	// EIP-2929
	var accessCost uint64
	if i.statedb.AddressInAccessList(addr) {
		accessCost = core.GasWarmStorageRead
	} else {
		accessCost = core.GasColdAccountAccess
		i.statedb.AddAddressToAccessList(addr)
	}

	// Adjust for already paid base cost
	baseCost := core.GasTable[core.CALLCODE]
	if accessCost > baseCost {
		callCost += (accessCost - baseCost)
	} else {
		i.gas += (baseCost - accessCost)
	}

	if value.Sign() > 0 {
		callCost += 9000 // GasCallValue
	}
	if i.gas < callCost {
		i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, callCost)
		i.reverted = true
		return
	}
	i.gas -= callCost

	// Snapshot state before call
	snapshot := i.statedb.Snapshot()

	// Calculate memory expansion
	if argsLength > 0 {
		if !i.consumeMemoryExpansion(argsOffset, argsLength) {
			i.statedb.RevertToSnapshot(snapshot)
			return
		}
	}
	if retLength > 0 {
		if !i.consumeMemoryExpansion(retOffset, retLength) {
			i.statedb.RevertToSnapshot(snapshot)
			return
		}
	}

	// Get code from target address but execute in caller's context
	code := i.statedb.GetCode(addr)
	args := i.memory.Read(argsOffset, argsLength)

	// Execute in caller's context (address stays as i.address)
	contract := NewWithCallData(code, args, i.statedb, i.address)
	contract.inheritExecutionContext(i)
	contract.SetBlockNumber(i.blockNumber)
	contract.SetTimestamp(i.timestamp)
	contract.SetCoinbase(i.coinbase)
	contract.SetBlockGasLimit(i.gasLimit)

	// Handle gas passing (EIP-150)
	gasLimit := gas.Uint64()
	available := i.gas
	cap := available - available/64
	if gasLimit > cap {
		gasLimit = cap
	}
	i.gas -= gasLimit

	// Add call stipend if value is transferred
	if value.Sign() > 0 {
		gasLimit += core.GasCallStipend
	}

	contract.SetGas(gasLimit)

	contract.SetCaller(i.address)
	contract.SetOrigin(i.origin)
	contract.SetCallValue(value)
	contract.SetChainID(i.chainID)
	contract.SetGasPrice(i.gasPrice)

	contract.Run()

	ret := contract.ReturnedCode()
	i.returnData = ret

	// Handle errors and revert
	if contract.Err() != nil {
		// Error (not clean revert): consume all gas
		i.statedb.RevertToSnapshot(snapshot)
		i.stack.PushSafe(big.NewInt(0))
		// Copy return data even on failure
		toCopy := uint64(len(ret))
		if toCopy > retLength {
			toCopy = retLength
		}
		if toCopy > 0 {
			i.memory.Write(retOffset, ret[:toCopy])
		}
	} else if contract.IsReverted() {
		// Clean revert (REVERT opcode): return remaining gas
		i.gas += contract.Gas()
		i.statedb.RevertToSnapshot(snapshot)
		i.stack.PushSafe(big.NewInt(0))
		// Copy return data even on failure
		toCopy := uint64(len(ret))
		if toCopy > retLength {
			toCopy = retLength
		}
		if toCopy > 0 {
			i.memory.Write(retOffset, ret[:toCopy])
		}
	} else {
		// Success: return remaining gas
		i.gas += contract.Gas()
		toCopy := uint64(len(ret))
		if toCopy > retLength {
			toCopy = retLength
		}
		if toCopy > 0 {
			i.memory.Write(retOffset, ret[:toCopy])
		}
		i.stack.PushSafe(big.NewInt(1))
	}
}

// opDelegateCall implements the DELEGATECALL opcode.
// Like CALLCODE but also preserves msg.sender and msg.value.
func opDelegateCall(i *Interpreter, _ byte) {
	// Stack: gas, addr, argsOffset, argsLength, retOffset, retLength (no value)
	gas := i.stack.PopSafe()
	addrBig := i.stack.PopSafe()
	addr := common.BigToAddress(addrBig)
	argsOffset := i.stack.PopSafe().Uint64()
	argsLength := i.stack.PopSafe().Uint64()
	retOffset := i.stack.PopSafe().Uint64()
	retLength := i.stack.PopSafe().Uint64()

	// Dynamic gas
	var callCost uint64

	// EIP-2929
	var accessCost uint64
	if i.statedb.AddressInAccessList(addr) {
		accessCost = core.GasWarmStorageRead
	} else {
		accessCost = core.GasColdAccountAccess
		i.statedb.AddAddressToAccessList(addr)
	}

	// Adjust for already paid base cost
	baseCost := core.GasTable[core.DELEGATECALL]
	if accessCost > baseCost {
		callCost += (accessCost - baseCost)
	} else {
		i.gas += (baseCost - accessCost)
	}

	if i.gas < callCost {
		i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, callCost)
		i.reverted = true
		return
	}
	i.gas -= callCost

	// Snapshot state before call
	snapshot := i.statedb.Snapshot()

	// Calculate memory expansion
	if argsLength > 0 {
		if !i.consumeMemoryExpansion(argsOffset, argsLength) {
			i.statedb.RevertToSnapshot(snapshot)
			return
		}
	}
	if retLength > 0 {
		if !i.consumeMemoryExpansion(retOffset, retLength) {
			i.statedb.RevertToSnapshot(snapshot)
			return
		}
	}

	// Get code from target address but execute in caller's context
	code := i.statedb.GetCode(addr)
	args := i.memory.Read(argsOffset, argsLength)

	// Execute in caller's context, preserving caller and value
	contract := NewWithCallData(code, args, i.statedb, i.address)
	contract.inheritExecutionContext(i)
	contract.SetBlockNumber(i.blockNumber)
	contract.SetTimestamp(i.timestamp)
	contract.SetCoinbase(i.coinbase)
	contract.SetBlockGasLimit(i.gasLimit)

	// Handle gas passing (EIP-150)
	gasLimit := gas.Uint64()
	available := i.gas
	cap := available - available/64
	if gasLimit > cap {
		gasLimit = cap
	}
	i.gas -= gasLimit
	contract.SetGas(gasLimit)

	contract.SetCaller(i.caller) // Preserve original caller
	contract.SetOrigin(i.origin)
	contract.SetCallValue(i.callvalue) // Preserve original value
	contract.SetChainID(i.chainID)
	contract.SetGasPrice(i.gasPrice)

	contract.Run()

	ret := contract.ReturnedCode()
	i.returnData = ret

	// Handle errors and revert
	if contract.Err() != nil {
		// Error (not clean revert): consume all gas
		i.statedb.RevertToSnapshot(snapshot)
		i.stack.PushSafe(big.NewInt(0))
		// Copy return data even on failure
		toCopy := uint64(len(ret))
		if toCopy > retLength {
			toCopy = retLength
		}
		if toCopy > 0 {
			i.memory.Write(retOffset, ret[:toCopy])
		}
	} else if contract.IsReverted() {
		// Clean revert (REVERT opcode): return remaining gas
		i.gas += contract.Gas()
		i.statedb.RevertToSnapshot(snapshot)
		i.stack.PushSafe(big.NewInt(0))
		// Copy return data even on failure
		toCopy := uint64(len(ret))
		if toCopy > retLength {
			toCopy = retLength
		}
		if toCopy > 0 {
			i.memory.Write(retOffset, ret[:toCopy])
		}
	} else {
		// Success: return remaining gas
		i.gas += contract.Gas()
		toCopy := uint64(len(ret))
		if toCopy > retLength {
			toCopy = retLength
		}
		if toCopy > 0 {
			i.memory.Write(retOffset, ret[:toCopy])
		}
		i.stack.PushSafe(big.NewInt(1))
	}
}

// opStaticCall implements the STATICCALL opcode.
// Like CALL but state modifications are not allowed.
func opStaticCall(i *Interpreter, _ byte) {
	// Stack: gas, addr, argsOffset, argsLength, retOffset, retLength (no value)
	gas := i.stack.PopSafe()
	addrBig := i.stack.PopSafe()
	addr := common.BigToAddress(addrBig)
	argsOffset := i.stack.PopSafe().Uint64()
	argsLength := i.stack.PopSafe().Uint64()
	retOffset := i.stack.PopSafe().Uint64()
	retLength := i.stack.PopSafe().Uint64()

	// Dynamic gas
	var callCost uint64

	// EIP-2929
	var accessCost uint64
	if i.statedb.AddressInAccessList(addr) {
		accessCost = core.GasWarmStorageRead
	} else {
		accessCost = core.GasColdAccountAccess
		i.statedb.AddAddressToAccessList(addr)
	}

	// Adjust for already paid base cost
	baseCost := core.GasTable[core.STATICCALL]
	if accessCost > baseCost {
		callCost += (accessCost - baseCost)
	} else {
		i.gas += (baseCost - accessCost)
	}

	if i.gas < callCost {
		i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, callCost)
		i.reverted = true
		return
	}
	i.gas -= callCost

	// Snapshot state before call
	snapshot := i.statedb.Snapshot()

	// Calculate memory expansion
	if argsLength > 0 {
		if !i.consumeMemoryExpansion(argsOffset, argsLength) {
			i.statedb.RevertToSnapshot(snapshot)
			return
		}
	}
	if retLength > 0 {
		if !i.consumeMemoryExpansion(retOffset, retLength) {
			i.statedb.RevertToSnapshot(snapshot)
			return
		}
	}

	code := i.statedb.GetCode(addr)
	args := i.memory.Read(argsOffset, argsLength)

	// Handle gas passing (EIP-150)
	gasLimit := gas.Uint64()
	available := i.gas
	cap := available - available/64
	if gasLimit > cap {
		gasLimit = cap
	}
	i.gas -= gasLimit

	// Check for precompiled contracts
	if IsPrecompiled(addr) {
		ret, remainingGas, err := RunPrecompiled(addr, args, gasLimit)
		i.returnData = ret
		i.gas += remainingGas

		if err != nil {
			i.statedb.RevertToSnapshot(snapshot)
			i.stack.PushSafe(big.NewInt(0))
		} else {
			// Copy return data to memory
			toCopy := uint64(len(ret))
			if toCopy > retLength {
				toCopy = retLength
			}
			if toCopy > 0 {
				i.memory.Write(retOffset, ret[:toCopy])
			}
			i.stack.PushSafe(big.NewInt(1))
		}
		return
	}

	// Execute regular contract (read-only)
	contract := NewWithCallData(code, args, i.statedb, addr)
	contract.inheritExecutionContext(i)
	contract.readOnly = true
	contract.SetBlockNumber(i.blockNumber)
	contract.SetTimestamp(i.timestamp)
	contract.SetCoinbase(i.coinbase)
	contract.SetBlockGasLimit(i.gasLimit)
	contract.SetGas(gasLimit)

	contract.SetCaller(i.address)
	contract.SetOrigin(i.origin)
	contract.SetCallValue(big.NewInt(0))
	contract.SetChainID(i.chainID)
	contract.SetGasPrice(i.gasPrice)

	contract.Run()

	ret := contract.ReturnedCode()
	i.returnData = ret

	// Handle errors and revert
	if contract.Err() != nil {
		// Error (not clean revert): consume all gas
		i.statedb.RevertToSnapshot(snapshot)
		i.stack.PushSafe(big.NewInt(0))
		// Copy return data even on failure
		toCopy := uint64(len(ret))
		if toCopy > retLength {
			toCopy = retLength
		}
		if toCopy > 0 {
			i.memory.Write(retOffset, ret[:toCopy])
		}
	} else if contract.IsReverted() {
		// Clean revert (REVERT opcode): return remaining gas
		i.gas += contract.Gas()
		i.statedb.RevertToSnapshot(snapshot)
		i.stack.PushSafe(big.NewInt(0))
		// Copy return data even on failure
		toCopy := uint64(len(ret))
		if toCopy > retLength {
			toCopy = retLength
		}
		if toCopy > 0 {
			i.memory.Write(retOffset, ret[:toCopy])
		}
	} else {
		// Success: return remaining gas
		i.gas += contract.Gas()
		toCopy := uint64(len(ret))
		if toCopy > retLength {
			toCopy = retLength
		}
		if toCopy > 0 {
			i.memory.Write(retOffset, ret[:toCopy])
		}
		i.stack.PushSafe(big.NewInt(1))
	}
}

// opSelfDestruct implements the SELFDESTRUCT opcode.
// Transfers all balance to the target and marks the contract for destruction.
func opSelfDestruct(i *Interpreter, _ byte) {
	if i.rejectWriteProtection() {
		return
	}
	addrBig := i.stack.PopSafe()
	addr := common.BigToAddress(addrBig)

	// Base cost (5000) is already paid by interpreter
	// EIP-2929: Additional cold access cost for beneficiary
	var cost uint64
	if !i.statedb.AddressInAccessList(addr) {
		cost += core.GasColdAccountAccess
		i.statedb.AddAddressToAccessList(addr)
	}
	// If warm, no additional cost (base 5000 already paid)

	// Dynamic gas: cost of creating new account
	balance := new(big.Int).Set(i.statedb.GetBalance(i.address))
	if balance.Sign() > 0 && !i.statedb.Exist(addr) {
		cost += 25000
	}

	if i.gas < cost {
		i.err = fmt.Errorf("out of gas: have %d, want %d", i.gas, cost)
		i.reverted = true
		return
	}
	i.gas -= cost

	// Transfer all balance
	if balance.Sign() > 0 {
		i.statedb.SubBalance(i.address, balance)
		i.statedb.AddBalance(addr, balance)
	}

	// Mark as suicided
	// EIP-6780: SELFDESTRUCT only clears the account if it is created in the same transaction.
	if i.statedb.HasBeenCreatedInCurrentTx(i.address) {
		i.statedb.Suicide(i.address)
	}
}
