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

	// Calculate Intrinsic Gas
	intrinsicGas := uint64(21000)
	if tx.To() == nil {
		intrinsicGas = 53000 // Contract creation
	}
	data := tx.Data()
	for _, b := range data {
		if b == 0 {
			intrinsicGas += 4
		} else {
			intrinsicGas += 16
		}
	}

	// Add Access List intrinsic gas
	if accessList := tx.AccessList(); accessList != nil {
		for _, entry := range accessList {
			intrinsicGas += 2400
			intrinsicGas += uint64(len(entry.StorageKeys)) * 1900
		}
	}

	if gas < intrinsicGas {
		return nil, 0, false, fmt.Errorf("intrinsic gas too low: have %d, want %d", gas, intrinsicGas)
	}

	// 3. Increment nonce
	statedb.SetNonce(sender, nonce+1)

	// Snapshot state before execution (for revert)
	snapshot := statedb.Snapshot()

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

	// Pre-warm Access List (EIP-2929)
	statedb.AddAddressToAccessList(sender)
	if to != nil {
		statedb.AddAddressToAccessList(*to)
	} else {
		statedb.AddAddressToAccessList(contractAddr)
	}
	// Precompiles (0x01 - 0x09) + 0x0A (Cancun)
	for i := 1; i <= 10; i++ {
		statedb.AddAddressToAccessList(common.BytesToAddress([]byte{byte(i)}))
	}
	// Add explicit Access List
	if accessList := tx.AccessList(); accessList != nil {
		for _, entry := range accessList {
			statedb.AddAddressToAccessList(entry.Address)
			for _, key := range entry.StorageKeys {
				statedb.AddSlotToAccessList(entry.Address, key)
			}
		}
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
		intr.SetBlockGasLimit(ctx.GasLimit)
		intr.SetGas(gas - intrinsicGas)
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

		if intr.Err() != nil {
			statedb.RevertToSnapshot(snapshot)
		} else if reverted {
			statedb.RevertToSnapshot(snapshot)
		} else {
			statedb.SetCode(contractAddr, ret)
		}
	} else {
		// Call
		code := statedb.GetCode(*to)
		intr = NewWithCallData(code, tx.Data(), statedb, *to)
		configureInterpreter(intr)

		intr.Run()
		ret = intr.ReturnedCode()
		reverted = intr.IsReverted()

		if intr.Err() != nil || reverted {
			statedb.RevertToSnapshot(snapshot)
		}
	}

	// Calculate Gas Used
	gasRemaining := intr.Gas()
	if intr.Err() != nil {
		gasRemaining = 0
	}
	gasUsed := gas - gasRemaining

	// Apply refund counter
	refund := statedb.GetRefund()
	maxRefund := gasUsed / 5 // London: /5. Before: /2.
	if refund > maxRefund {
		refund = maxRefund
	}
	gasRemaining += refund
	gasUsed -= refund

	// Refund unused gas
	refundEth := new(big.Int).Mul(big.NewInt(int64(gasRemaining)), gasPrice)
	statedb.AddBalance(sender, refundEth)

	// Pay Miner
	// EffectiveTip = GasPrice - BaseFee
	effectiveTip := new(big.Int).Set(gasPrice)
	if ctx.BaseFee != nil {
		effectiveTip.Sub(effectiveTip, ctx.BaseFee)
		if effectiveTip.Sign() < 0 {
			effectiveTip.SetInt64(0)
		}
	}

	minerReward := new(big.Int).Mul(big.NewInt(int64(gasUsed)), effectiveTip)
	statedb.AddBalance(ctx.Coinbase, minerReward)

	return ret, gasUsed, reverted, nil
}
