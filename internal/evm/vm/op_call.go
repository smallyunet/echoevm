package vm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"golang.org/x/crypto/sha3"
)

// opCreate implements the CREATE opcode.
func opCreate(i *Interpreter, _ byte) {
	// Stack: value, offset, length
	value := i.stack.PopSafe()
	offset := i.stack.PopSafe().Uint64()
	length := i.stack.PopSafe().Uint64()

	// 1. Check balance
	if i.statedb.GetBalance(i.address).Cmp(value) < 0 {
		i.stack.PushSafe(big.NewInt(0))
		return
	}

	// 2. Calculate new address
	nonce := i.statedb.GetNonce(i.address)
	i.statedb.SetNonce(i.address, nonce+1)
	addr := crypto.CreateAddress(i.address, nonce)

	// 3. Create account and transfer value
	// If account already exists (collision), it should fail (return 0).
	if i.statedb.Exist(addr) {
		if i.statedb.GetNonce(addr) != 0 || i.statedb.GetCodeSize(addr) != 0 {
			i.stack.PushSafe(big.NewInt(0))
			return
		}
	}
	i.statedb.CreateAccount(addr)
	i.statedb.SetNonce(addr, 1) // EIP-161: New accounts have nonce 1
	i.statedb.SubBalance(i.address, value)
	i.statedb.AddBalance(addr, value)

	if !i.consumeMemoryExpansion(offset, length) {
		return
	}

	// 4. Get init code
	initCode := i.memory.Read(offset, length)

	// 5. Execute init code
	contract := New(initCode, i.statedb, addr)
	contract.SetBlockNumber(i.blockNumber)
	contract.SetTimestamp(i.timestamp)
	contract.SetCoinbase(i.coinbase)
	contract.SetBlockGasLimit(i.gasLimit)
	
	// EIP-150: 63/64 rule
	available := i.gas
	gasLimit := available - available/64
	i.gas -= gasLimit
	contract.SetGas(gasLimit)
	
	contract.SetCaller(i.address)
	contract.SetOrigin(i.origin)
	contract.SetCallValue(value)

	contract.Run()

	if contract.IsReverted() || contract.Err() != nil {
		i.stack.PushSafe(big.NewInt(0))
		return
	}

	// 6. Set code
	ret := contract.ReturnedCode()
	i.statedb.SetCode(addr, ret)

	// 7. Push address
	i.stack.PushSafe(addr.Big())
}

// opCreate2 implements the CREATE2 opcode (EIP-1014)
func opCreate2(i *Interpreter, _ byte) {
	// Stack: value, offset, length, salt
	value := i.stack.PopSafe()
	offset := i.stack.PopSafe().Uint64()
	length := i.stack.PopSafe().Uint64()
	salt := i.stack.PopSafe()

	// 1. Check balance
	if i.statedb.GetBalance(i.address).Cmp(value) < 0 {
		i.stack.PushSafe(big.NewInt(0))
		return
	}

	if !i.consumeMemoryExpansion(offset, length) {
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

	// 4. Check for collision
	if i.statedb.Exist(addr) {
		if i.statedb.GetNonce(addr) != 0 || i.statedb.GetCodeSize(addr) != 0 {
			i.stack.PushSafe(big.NewInt(0))
			return
		}
	}

	// 5. Increment nonce and create account
	nonce := i.statedb.GetNonce(i.address)
	i.statedb.SetNonce(i.address, nonce+1)
	i.statedb.CreateAccount(addr)
	i.statedb.SetNonce(addr, 1)
	i.statedb.SubBalance(i.address, value)
	i.statedb.AddBalance(addr, value)

	// 6. Execute init code
	contract := New(initCode, i.statedb, addr)
	contract.SetBlockNumber(i.blockNumber)
	contract.SetTimestamp(i.timestamp)
	contract.SetCoinbase(i.coinbase)
	contract.SetBlockGasLimit(i.gasLimit)
	
	// EIP-150: 63/64 rule
	available := i.gas
	gasLimit := available - available/64
	i.gas -= gasLimit
	contract.SetGas(gasLimit)
	
	contract.SetCaller(i.address)
	contract.SetOrigin(i.origin)
	contract.SetCallValue(value)

	contract.Run()

	if contract.IsReverted() || contract.Err() != nil {
		i.stack.PushSafe(big.NewInt(0))
		return
	}

	// 7. Set code and push address
	ret := contract.ReturnedCode()
	i.statedb.SetCode(addr, ret)
	i.stack.PushSafe(addr.Big())
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

	// Dynamic gas
	var callCost uint64
	
	// EIP-2929
	var accessCost uint64
	if i.statedb.AddressInAccessList(addr) {
		accessCost = 100 // GasWarmStorageRead
	} else {
		accessCost = 2600 // GasColdAccountAccess
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

	// 5. Execute
	contract := NewWithCallData(code, args, i.statedb, addr)
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
		gasLimit += 2300 // GasCallStipend
	}
	
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
		accessCost = 100 // GasWarmStorageRead
	} else {
		accessCost = 2600 // GasColdAccountAccess
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
		gasLimit += 2300 // GasCallStipend
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
		accessCost = 100 // GasWarmStorageRead
	} else {
		accessCost = 2600 // GasColdAccountAccess
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
		accessCost = 100 // GasWarmStorageRead
	} else {
		accessCost = 2600 // GasColdAccountAccess
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

	// Note: A proper implementation would use a read-only state wrapper
	// For now we execute normally but with value = 0
	contract := NewWithCallData(code, args, i.statedb, addr)
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
	addrBig := i.stack.PopSafe()
	addr := common.BigToAddress(addrBig)

	// Base cost (5000) is already paid by interpreter
	// EIP-2929: Additional cold access cost for beneficiary
	var cost uint64
	if !i.statedb.AddressInAccessList(addr) {
		cost += 2600 // GasColdAccountAccess
		i.statedb.AddAddressToAccessList(addr)
	}
	// If warm, no additional cost (base 5000 already paid)

	// Dynamic gas: cost of creating new account
	balance := i.statedb.GetBalance(i.address)
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
	i.statedb.Suicide(i.address)
}

// keccak256 helper for CREATE2
func keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}
