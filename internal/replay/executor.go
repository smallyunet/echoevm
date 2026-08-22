package replay

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/smallyunet/echoevm/internal/differential"
	"github.com/smallyunet/echoevm/internal/evm/core"
	explaintrace "github.com/smallyunet/echoevm/internal/trace"
)

// ReplayWitness executes a self-contained witness exclusively with EchoEVM.
// It never contacts an RPC endpoint or compares against another EVM engine.
func ReplayWitness(ctx context.Context, req ReplayRequest) (ReplayResult, error) {
	if err := req.Witness.Validate(); err != nil {
		return ReplayResult{}, err
	}
	if req.Profile != "" {
		if err := explaintrace.ValidateEvidenceProfile(req.Profile); err != nil {
			return ReplayResult{}, err
		}
		if req.Limit < 0 || req.MaxMemoryBytes < 0 {
			return ReplayResult{}, errors.New("evidence limits must be non-negative")
		}
		if req.MaxMemoryBytes == 0 {
			req.MaxMemoryBytes = DefaultEvidenceMemoryBytes
		}
	}

	var tx types.Transaction
	if err := tx.UnmarshalBinary(req.Witness.Transaction); err != nil {
		return ReplayResult{}, fmt.Errorf("decode replay witness transaction: %w", err)
	}
	signer := types.MakeSigner(params.MainnetChainConfig, req.Witness.Header.Number, req.Witness.Header.Time)
	sender, err := types.Sender(signer, &tx)
	if err != nil {
		return ReplayResult{}, fmt.Errorf("recover replay transaction sender: %w", err)
	}
	prestate, err := witnessPrestate(req.Witness.Prestate)
	if err != nil {
		return ReplayResult{}, err
	}
	if _, ok := prestate[sender]; !ok {
		return ReplayResult{}, fmt.Errorf("replay witness is missing sender account %s", sender.Hex())
	}
	blockHashes, err := witnessBlockHashes(req.Witness.BlockHashes)
	if err != nil {
		return ReplayResult{}, err
	}
	execution, state, events, err := runEcho(ctx, &tx, sender, req.Witness.ChainID, &req.Witness.Header, prestate, blockHashes, req.Profile != "", req.MaxMemoryBytes)
	if err != nil {
		return ReplayResult{}, err
	}
	provenance, err := req.Witness.Provenance()
	if err != nil {
		return ReplayResult{}, err
	}
	transaction := summarizeExecution(&tx, sender, req.Witness, execution.Status, execution.GasUsed)
	result := ReplayResult{
		Transaction: transaction,
		Execution:   execution,
		State:       flattenState(state),
		Warnings:    replayWarnings(req.Witness.Header.Time, req.Witness.Header.Number.Uint64(), execution, blockHashes),
		Witness:     provenance,
	}
	if req.Profile != "" {
		document, evidenceErr := explaintrace.BuildEvidence(explaintrace.ExecutionResult{
			Status: string(execution.Status), GasLimit: tx.Gas(), GasUsed: execution.GasUsed,
			ReturnData: execution.ReturnData, TotalSteps: len(events), ExecutionError: execution.Error,
		}, events, req.Profile, req.Limit)
		if evidenceErr != nil {
			return ReplayResult{}, evidenceErr
		}
		result.Evidence = &ReplayEvidenceResult{
			EvidenceDocument: document,
			Transaction:      transaction,
			Witness:          provenance,
			Warnings:         append([]string(nil), result.Warnings...),
		}
	}
	return result, nil
}

func witnessPrestate(accounts map[string]WitnessAccount) (map[common.Address]prestateAccount, error) {
	state := make(map[common.Address]prestateAccount, len(accounts))
	for rawAddress, account := range accounts {
		if !common.IsHexAddress(rawAddress) {
			return nil, fmt.Errorf("replay witness contains invalid address %q", rawAddress)
		}
		state[common.HexToAddress(rawAddress)] = prestateAccount{
			Balance: account.Balance,
			Nonce:   account.Nonce,
			Code:    append(hexutil.Bytes(nil), account.Code...),
			Storage: cloneStorage(account.Storage),
		}
	}
	return state, nil
}

func witnessBlockHashes(values map[string]common.Hash) (map[uint64]common.Hash, error) {
	hashes := make(map[uint64]common.Hash, len(values))
	for rawNumber, hash := range values {
		number, err := strconv.ParseUint(rawNumber, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("replay witness contains invalid blockHashes key %q", rawNumber)
		}
		hashes[number] = hash
	}
	return hashes, nil
}

func cloneStorage(storage map[common.Hash]common.Hash) map[common.Hash]common.Hash {
	if storage == nil {
		return nil
	}
	cloned := make(map[common.Hash]common.Hash, len(storage))
	for key, value := range storage {
		cloned[key] = value
	}
	return cloned
}

func summarizeExecution(tx *types.Transaction, sender common.Address, witness Witness, status differential.Status, gasUsed uint64) TransactionSummary {
	var to *string
	if tx.To() != nil {
		value := tx.To().Hex()
		to = &value
	}
	return TransactionSummary{
		Hash: tx.Hash().Hex(), ExplorerURL: explorerURL(tx.Hash()), ChainID: witness.ChainID,
		BlockNumber: witness.Header.Number.Uint64(), BlockHash: witness.BlockHash.Hex(),
		Index: witness.TransactionIndex, From: sender.Hex(), To: to, Value: tx.Value().String(),
		GasLimit: tx.Gas(), GasUsed: gasUsed, Type: tx.Type(), Input: hexutil.Encode(tx.Data()), Status: string(status),
		Fork: forkName(witness.Header.Time),
	}
}

func flattenState(state *core.MemoryStateDB) map[string]string {
	result := make(map[string]string)
	state.ForEachAccount(func(address common.Address) bool {
		prefix := strings.ToLower(address.Hex())
		result[prefix+":balance"] = state.GetBalance(address).String()
		result[prefix+":nonce"] = strconv.FormatUint(state.GetNonce(address), 10)
		if code := state.GetCode(address); len(code) > 0 {
			result[prefix+":code"] = "0x" + hex.EncodeToString(code)
		}
		state.ForEachStorage(address, func(key, value common.Hash) bool {
			result[prefix+":storage:"+key.Hex()] = value.Hex()
			return true
		})
		return true
	})
	return result
}
