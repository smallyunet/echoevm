package vm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

// ApplyTransaction attempts to apply a transaction to the given state database.
// It handles gas deduction, nonce increment, value transfer, and VM execution.
func ApplyTransaction(
	statedb core.StateDB,
	tx *types.Transaction,
	sender common.Address,
	blockNumber *big.Int,
	timestamp uint64,
	coinbase common.Address,
	gasLimit uint64,
) ([]byte, uint64, bool, error) {

	// 1. Check nonce
	nonce := statedb.GetNonce(sender)
	if nonce != tx.Nonce() {
		return nil, 0, false, fmt.Errorf("nonce mismatch: expected %d, got %d", nonce, tx.Nonce())
	}

	// 2. Buy gas
	gas := tx.Gas()
	gasPrice := tx.GasPrice()
	cost := new(big.Int).Mul(big.NewInt(int64(gas)), gasPrice)
	if statedb.GetBalance(sender).Cmp(cost) < 0 {
		return nil, 0, false, fmt.Errorf("insufficient funds for gas: have %v, want %v", statedb.GetBalance(sender), cost)
	}
	statedb.SubBalance(sender, cost)

	// 3. Increment nonce
	statedb.SetNonce(sender, nonce+1)

	// 4. Value transfer (if any) and execution
	value := tx.Value()
	to := tx.To()
	var ret []byte
	var reverted bool

	// Calculate contract address if creation
	var contractAddr common.Address
	if to == nil {
		contractAddr = crypto.CreateAddress(sender, nonce)
	}

	// Transfer value
	if value.Sign() > 0 {
		if statedb.GetBalance(sender).Cmp(value) < 0 {
			return nil, 0, false, fmt.Errorf("insufficient funds for transfer: have %v, want %v", statedb.GetBalance(sender), value)
		}
		statedb.SubBalance(sender, value)
		if to != nil {
			statedb.AddBalance(*to, value)
		} else {
			statedb.AddBalance(contractAddr, value)
		}
	}

	// Create Interpreter
	var intr *Interpreter
	if to == nil {
		// Contract Creation
		intr = New(tx.Data(), statedb, contractAddr)

		// Run to get runtime code
		intr.SetBlockNumber(blockNumber.Uint64())
		intr.SetTimestamp(timestamp)
		intr.SetCoinbase(coinbase)
		intr.SetGasLimit(gasLimit)
		intr.Run()

		ret = intr.ReturnedCode()
		reverted = intr.IsReverted()

		if !reverted {
			statedb.SetCode(contractAddr, ret)
		}
		return ret, gas, reverted, nil
	} else {
		// Call
		code := statedb.GetCode(*to)
		intr = NewWithCallData(code, tx.Data(), statedb, *to)

		intr.SetBlockNumber(blockNumber.Uint64())
		intr.SetTimestamp(timestamp)
		intr.SetCoinbase(coinbase)
		intr.SetGasLimit(gasLimit)

		intr.Run()
		ret = intr.ReturnedCode()
		reverted = intr.IsReverted()

		return ret, gas, reverted, nil
	}
}
