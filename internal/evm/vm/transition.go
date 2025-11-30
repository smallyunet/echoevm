package vm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

// BlockContext contains block-level context for transaction execution.
type BlockContext struct {
	BlockNumber *big.Int
	Timestamp   uint64
	Coinbase    common.Address
	GasLimit    uint64
	BaseFee     *big.Int
	Difficulty  *big.Int
	Random      *big.Int // PREVRANDAO for post-merge
	ChainID     *big.Int
}

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
	ctx := &BlockContext{
		BlockNumber: blockNumber,
		Timestamp:   timestamp,
		Coinbase:    coinbase,
		GasLimit:    gasLimit,
		ChainID:     big.NewInt(1), // default mainnet
	}
	return ApplyTransactionWithContext(statedb, tx, sender, ctx)
}

// ApplyTransactionWithContext applies a transaction with full block context.
func ApplyTransactionWithContext(
	statedb core.StateDB,
	tx *types.Transaction,
	sender common.Address,
	ctx *BlockContext,
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

	// Helper to configure interpreter with block context
	configureInterpreter := func(intr *Interpreter) {
		if ctx.BlockNumber != nil {
			intr.SetBlockNumber(ctx.BlockNumber.Uint64())
		}
		intr.SetTimestamp(ctx.Timestamp)
		intr.SetCoinbase(ctx.Coinbase)
		intr.SetGasLimit(ctx.GasLimit)
		intr.SetCaller(sender)
		intr.SetOrigin(sender)
		intr.SetCallValue(value)
		intr.SetGasPrice(gasPrice)
		if ctx.BaseFee != nil {
			intr.SetBaseFee(ctx.BaseFee)
		}
		if ctx.Difficulty != nil {
			intr.SetDifficulty(ctx.Difficulty)
		}
		if ctx.Random != nil {
			intr.SetRandom(ctx.Random)
		}
		if ctx.ChainID != nil {
			intr.SetChainID(ctx.ChainID)
		}
	}

	// Create Interpreter
	var intr *Interpreter
	if to == nil {
		// Contract Creation
		intr = New(tx.Data(), statedb, contractAddr)
		configureInterpreter(intr)

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
		configureInterpreter(intr)

		intr.Run()
		ret = intr.ReturnedCode()
		reverted = intr.IsReverted()

		return ret, gas, reverted, nil
	}
}
