package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
	if i.statedb.GetBalance(i.address).Cmp(value) < 0 {
		i.stack.PushSafe(big.NewInt(0))
		return
	}
	i.statedb.SubBalance(i.address, value)
	i.statedb.AddBalance(addr, value)

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

	contract.Run()

	// 5. Handle return data
	ret := contract.ReturnedCode()
	toCopy := uint64(len(ret))
	if toCopy > retLength {
		toCopy = retLength
	}
	if toCopy > 0 {
		i.memory.Write(retOffset, ret[:toCopy])
	}

	// 6. Push result
	if contract.IsReverted() || contract.Err() != nil {
		i.stack.PushSafe(big.NewInt(0))
	} else {
		i.stack.PushSafe(big.NewInt(1))
	}
}

// opDelegateCall is a stub that always fails. It pops the expected
// arguments and pushes 0 to indicate failure.
func opDelegateCall(i *Interpreter, _ byte) {
	for n := 0; n < 6; n++ {
		if i.stack.Len() > 0 {
			i.stack.PopSafe()
		}
	}
	i.stack.PushSafe(big.NewInt(0))
}
