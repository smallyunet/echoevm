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
	BlobBaseFee *big.Int
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
	return ApplyTransactionWithContextAndHook(statedb, tx, sender, ctx, nil)
}

// ApplyTransactionWithContextAndHook applies a full transaction and emits a
// transaction-wide opcode trace, including nested CALL and CREATE frames.
func ApplyTransactionWithContextAndHook(
	statedb core.StateDB,
	tx *types.Transaction,
	sender common.Address,
	ctx *BlockContext,
	hook func(TraceStep) bool,
) ([]byte, uint64, bool, error) {
	// Validate the transaction before mutating state.
	nonce := statedb.GetNonce(sender)
	if nonce != tx.Nonce() {
		return nil, 0, false, fmt.Errorf("nonce mismatch: expected %d, got %d", nonce, tx.Nonce())
	}

	gas := tx.Gas()
	gasPrice := tx.GasPrice()
	if ctx.BaseFee != nil && tx.Type() != types.LegacyTxType && tx.Type() != types.AccessListTxType {
		gasPrice = new(big.Int).Add(ctx.BaseFee, tx.EffectiveGasTipValue(ctx.BaseFee))
	}
	value := tx.Value()
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
	if tx.To() == nil && len(data) > 0 {
		// EIP-3860 (Shanghai): charge two gas for each 32-byte initcode word.
		intrinsicGas += 2 * ((uint64(len(data)) + 31) / 32)
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

	gasCost := new(big.Int).Mul(new(big.Int).SetUint64(gas), gasPrice)
	blobCost := new(big.Int)
	if ctx.BlobBaseFee != nil && tx.BlobGas() > 0 {
		blobCost.Mul(new(big.Int).SetUint64(tx.BlobGas()), ctx.BlobBaseFee)
	}
	requiredBalance := new(big.Int).Add(new(big.Int).Set(gasCost), blobCost)
	requiredBalance.Add(requiredBalance, value)
	if statedb.GetBalance(sender).Cmp(requiredBalance) < 0 {
		return nil, 0, false, fmt.Errorf("insufficient funds: have %v, want %v", statedb.GetBalance(sender), requiredBalance)
	}

	statedb.PrepareTransaction()
	statedb.SubBalance(sender, gasCost)
	if blobCost.Sign() > 0 {
		statedb.SubBalance(sender, blobCost)
	}
	statedb.SetNonce(sender, nonce+1)

	// Gas purchase and nonce increment survive execution failure. State changes
	// after this snapshot are reverted on REVERT or exceptional halt.
	snapshot := statedb.Snapshot()

	to := tx.To()
	var ret []byte
	var reverted bool
	var executionErr error
	gasRemaining := gas - intrinsicGas

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
		intr.SetTraceContext(hook, 0)
		intr.SetBlobHashes(tx.BlobHashes())
		if ctx.BlobBaseFee != nil {
			intr.SetBlobBaseFee(ctx.BlobBaseFee)
		}
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

	if to != nil && IsPrecompiled(*to) {
		ret, gasRemaining, executionErr = RunPrecompiled(*to, tx.Data(), gasRemaining)
		if executionErr != nil {
			reverted = true
			gasRemaining = 0
			statedb.RevertToSnapshot(snapshot)
		}
	} else if to == nil {
		// Contract Creation
		intr := New(tx.Data(), statedb, contractAddr)
		configureInterpreter(intr)

		intr.Run()

		ret = intr.ReturnedCode()
		reverted = intr.IsReverted()
		executionErr = intr.Err()
		gasRemaining = intr.Gas()

		if executionErr != nil || reverted {
			statedb.RevertToSnapshot(snapshot)
		} else {
			statedb.SetCode(contractAddr, ret)
		}
	} else {
		// Call
		code := statedb.GetCode(*to)
		intr := NewWithCallData(code, tx.Data(), statedb, *to)
		configureInterpreter(intr)

		intr.Run()
		ret = intr.ReturnedCode()
		reverted = intr.IsReverted()
		executionErr = intr.Err()
		gasRemaining = intr.Gas()

		if executionErr != nil || reverted {
			statedb.RevertToSnapshot(snapshot)
		}
	}

	if executionErr != nil {
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
	refundEth := new(big.Int).Mul(new(big.Int).SetUint64(gasRemaining), gasPrice)
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

	minerReward := new(big.Int).Mul(new(big.Int).SetUint64(gasUsed), effectiveTip)
	statedb.AddBalance(ctx.Coinbase, minerReward)

	return ret, gasUsed, reverted, executionErr
}
