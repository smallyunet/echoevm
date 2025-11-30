package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

	// 4. Get init code
	initCode := i.memory.Read(offset, length)

	// 5. Execute init code
	contract := New(initCode, i.statedb, addr)
	contract.SetBlockNumber(i.blockNumber)
	contract.SetTimestamp(i.timestamp)
	contract.SetCoinbase(i.coinbase)
	contract.SetGasLimit(i.gasLimit)
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
	contract.SetGasLimit(i.gasLimit)
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

	// 1. Transfer value
	if value.Sign() > 0 && i.statedb.GetBalance(i.address).Cmp(value) < 0 {
		i.stack.PushSafe(big.NewInt(0))
		i.returnData = nil
		return
	}
	if value.Sign() > 0 {
		i.statedb.SubBalance(i.address, value)
		i.statedb.AddBalance(addr, value)
	}

	// 2. Get code
	code := i.statedb.GetCode(addr)

	// 3. Get input data
	args := i.memory.Read(argsOffset, argsLength)

	// 4. Execute
	contract := NewWithCallData(code, args, i.statedb, addr)
	contract.SetBlockNumber(i.blockNumber)
	contract.SetTimestamp(i.timestamp)
	contract.SetCoinbase(i.coinbase)
	contract.SetGasLimit(gas.Uint64())
	contract.SetCaller(i.address)
	contract.SetOrigin(i.origin)
	contract.SetCallValue(value)
	contract.SetChainID(i.chainID)
	contract.SetGasPrice(i.gasPrice)

	contract.Run()

	// 5. Store return data
	ret := contract.ReturnedCode()
	i.returnData = ret

	// 6. Copy to memory
	toCopy := uint64(len(ret))
	if toCopy > retLength {
		toCopy = retLength
	}
	if toCopy > 0 {
		i.memory.Write(retOffset, ret[:toCopy])
	}

	// 7. Push result
	if contract.IsReverted() || contract.Err() != nil {
		i.stack.PushSafe(big.NewInt(0))
	} else {
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

	// Get code from target address but execute in caller's context
	code := i.statedb.GetCode(addr)
	args := i.memory.Read(argsOffset, argsLength)

	// Execute in caller's context (address stays as i.address)
	contract := NewWithCallData(code, args, i.statedb, i.address)
	contract.SetBlockNumber(i.blockNumber)
	contract.SetTimestamp(i.timestamp)
	contract.SetCoinbase(i.coinbase)
	contract.SetGasLimit(gas.Uint64())
	contract.SetCaller(i.address)
	contract.SetOrigin(i.origin)
	contract.SetCallValue(value)
	contract.SetChainID(i.chainID)
	contract.SetGasPrice(i.gasPrice)

	contract.Run()

	ret := contract.ReturnedCode()
	i.returnData = ret

	toCopy := uint64(len(ret))
	if toCopy > retLength {
		toCopy = retLength
	}
	if toCopy > 0 {
		i.memory.Write(retOffset, ret[:toCopy])
	}

	if contract.IsReverted() || contract.Err() != nil {
		i.stack.PushSafe(big.NewInt(0))
	} else {
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

	// Get code from target address but execute in caller's context
	code := i.statedb.GetCode(addr)
	args := i.memory.Read(argsOffset, argsLength)

	// Execute in caller's context, preserving caller and value
	contract := NewWithCallData(code, args, i.statedb, i.address)
	contract.SetBlockNumber(i.blockNumber)
	contract.SetTimestamp(i.timestamp)
	contract.SetCoinbase(i.coinbase)
	contract.SetGasLimit(gas.Uint64())
	contract.SetCaller(i.caller) // Preserve original caller
	contract.SetOrigin(i.origin)
	contract.SetCallValue(i.callvalue) // Preserve original value
	contract.SetChainID(i.chainID)
	contract.SetGasPrice(i.gasPrice)

	contract.Run()

	ret := contract.ReturnedCode()
	i.returnData = ret

	toCopy := uint64(len(ret))
	if toCopy > retLength {
		toCopy = retLength
	}
	if toCopy > 0 {
		i.memory.Write(retOffset, ret[:toCopy])
	}

	if contract.IsReverted() || contract.Err() != nil {
		i.stack.PushSafe(big.NewInt(0))
	} else {
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

	code := i.statedb.GetCode(addr)
	args := i.memory.Read(argsOffset, argsLength)

	// Note: A proper implementation would use a read-only state wrapper
	// For now we execute normally but with value = 0
	contract := NewWithCallData(code, args, i.statedb, addr)
	contract.SetBlockNumber(i.blockNumber)
	contract.SetTimestamp(i.timestamp)
	contract.SetCoinbase(i.coinbase)
	contract.SetGasLimit(gas.Uint64())
	contract.SetCaller(i.address)
	contract.SetOrigin(i.origin)
	contract.SetCallValue(big.NewInt(0))
	contract.SetChainID(i.chainID)
	contract.SetGasPrice(i.gasPrice)

	contract.Run()

	ret := contract.ReturnedCode()
	i.returnData = ret

	toCopy := uint64(len(ret))
	if toCopy > retLength {
		toCopy = retLength
	}
	if toCopy > 0 {
		i.memory.Write(retOffset, ret[:toCopy])
	}

	if contract.IsReverted() || contract.Err() != nil {
		i.stack.PushSafe(big.NewInt(0))
	} else {
		i.stack.PushSafe(big.NewInt(1))
	}
}

// opSelfDestruct implements the SELFDESTRUCT opcode.
// Transfers all balance to the target and marks the contract for destruction.
func opSelfDestruct(i *Interpreter, _ byte) {
	addrBig := i.stack.PopSafe()
	addr := common.BigToAddress(addrBig)

	// Transfer all balance
	balance := i.statedb.GetBalance(i.address)
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
