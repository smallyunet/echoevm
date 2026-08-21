package replay

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/smallyunet/echoevm/internal/differential"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	explaintrace "github.com/smallyunet/echoevm/internal/trace"
)

const traceSemantics = "full transaction pre-op PC/opcode/gas/stack across nested frames; EchoEVM post-op values are captured directly while RPC reference steps are Geth struct logs"

type Service struct {
	rpc Caller
}

func NewService(ctx context.Context, rpcURL string) (*Service, error) {
	if strings.TrimSpace(rpcURL) == "" {
		return nil, errors.New("ethereum RPC URL is not configured")
	}
	client, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, NewError(ErrorUnavailable, fmt.Errorf("connect to Ethereum RPC: %w", err))
	}
	return &Service{rpc: client}, nil
}

func NewServiceWithCaller(caller Caller) *Service { return &Service{rpc: caller} }

func (s *Service) Replay(ctx context.Context, req Request) (Result, error) {
	if s == nil || s.rpc == nil {
		return Result{}, errors.New("transaction replay service is not configured")
	}
	if req.Profile != "" {
		if err := explaintrace.ValidateEvidenceProfile(req.Profile); err != nil {
			return Result{}, err
		}
		if req.Limit < 0 || req.MaxMemoryBytes < 0 {
			return Result{}, errors.New("evidence limits must be non-negative")
		}
		if req.MaxMemoryBytes == 0 {
			req.MaxMemoryBytes = DefaultEvidenceMemoryBytes
		}
	}
	ref, err := ParseTransactionReference(req.Input)
	if err != nil {
		return Result{}, err
	}
	chainID, err := s.chainID(ctx)
	if err != nil {
		return Result{}, err
	}
	if chainID != ethereumMainnetChainID {
		return Result{}, NewError(ErrorUnavailable, fmt.Errorf("configured RPC is chain %d; Ethereum Mainnet chain %d is required", chainID, ethereumMainnetChainID))
	}
	if ref.ChainID != chainID {
		return Result{}, fmt.Errorf("input targets chain %d but configured RPC is chain %d", ref.ChainID, chainID)
	}
	tx, meta, err := s.transaction(ctx, ref.Hash)
	if err != nil {
		return Result{}, err
	}
	var receipt types.Receipt
	if err := s.rpc.CallContext(ctx, &receipt, "eth_getTransactionReceipt", ref.Hash); err != nil {
		return Result{}, NewError(ErrorUpstream, fmt.Errorf("load transaction receipt: %w", err))
	}
	if receipt.TxHash == (common.Hash{}) {
		return Result{}, NewError(ErrorConflict, errors.New("transaction receipt is unavailable; the transaction may still be pending"))
	}
	var header types.Header
	if err := s.rpc.CallContext(ctx, &header, "eth_getBlockByHash", meta.BlockHash, false); err != nil {
		return Result{}, NewError(ErrorUpstream, fmt.Errorf("load transaction block: %w", err))
	}
	if header.Number == nil {
		return Result{}, NewError(ErrorUpstream, errors.New("RPC returned an incomplete transaction block"))
	}
	prestate, err := s.prestate(ctx, ref.Hash)
	if err != nil {
		return Result{}, err
	}
	reference, err := s.referenceTrace(ctx, ref.Hash, &receipt)
	if err != nil {
		return Result{}, err
	}
	diff, err := s.stateDiff(ctx, ref.Hash)
	if err != nil {
		return Result{}, err
	}
	echo, state, evidenceEvents, err := runEcho(ctx, tx, meta.From, chainID, &header, prestate, req.Profile != "", req.MaxMemoryBytes)
	if err != nil {
		return Result{}, err
	}
	result := compare(echo, reference)
	result.EchoState, result.GethState = compareState(state, diff)
	result.StateMatch = mapsEqual(result.EchoState, result.GethState)
	result.Match = result.Match && result.StateMatch
	if !result.StateMatch && result.FirstDivergence == nil {
		result.FirstDivergence = &differential.Divergence{Kind: "result", Field: "state", EchoEVM: result.EchoState, Geth: result.GethState, Description: "post-transaction state differs"}
	}
	result.Transaction = summarize(tx, meta, &receipt, &header, chainID)
	result.TraceSemantics = traceSemantics
	result.Warnings = replayWarnings(header.Time, tx, echo)
	if req.Profile != "" {
		document, evidenceErr := explaintrace.BuildEvidence(explaintrace.ExecutionResult{
			Status: string(echo.Status), GasLimit: tx.Gas(), GasUsed: echo.GasUsed,
			ReturnData: echo.ReturnData, TotalSteps: len(evidenceEvents), ExecutionError: echo.Error,
		}, evidenceEvents, req.Profile, req.Limit)
		if evidenceErr != nil {
			return Result{}, evidenceErr
		}
		result.Evidence = &EvidenceResult{
			EvidenceDocument: document,
			Transaction:      result.Transaction,
			Warnings:         append([]string(nil), result.Warnings...),
			Comparison: EvidenceComparison{
				Match: result.Match, StatusMatch: result.StatusMatch, ReturnDataMatch: result.ReturnDataMatch,
				GasMatch: result.GasMatch, StateMatch: result.StateMatch, TraceMatch: result.TraceMatch,
				FirstDivergence: result.FirstDivergence,
			},
		}
	}
	return result, nil
}

func (s *Service) chainID(ctx context.Context) (uint64, error) {
	var value hexutil.Uint64
	if err := s.rpc.CallContext(ctx, &value, "eth_chainId"); err != nil {
		return 0, NewError(ErrorUpstream, fmt.Errorf("load RPC chain ID: %w", err))
	}
	return uint64(value), nil
}

type rpcRecentBlock struct {
	Number       hexutil.Uint64 `json:"number"`
	Transactions []common.Hash  `json:"transactions"`
}

func (s *Service) RecentTransactions(ctx context.Context) (RecentTransactions, error) {
	if s == nil || s.rpc == nil {
		return RecentTransactions{}, errors.New("transaction replay service is not configured")
	}
	chainID, err := s.chainID(ctx)
	if err != nil {
		return RecentTransactions{}, err
	}
	if chainID != ethereumMainnetChainID {
		return RecentTransactions{}, NewError(ErrorUnavailable, fmt.Errorf("configured RPC is chain %d; Ethereum Mainnet chain %d is required", chainID, ethereumMainnetChainID))
	}
	var block rpcRecentBlock
	if err := s.rpc.CallContext(ctx, &block, "eth_getBlockByNumber", "latest", false); err != nil {
		return RecentTransactions{}, NewError(ErrorUpstream, fmt.Errorf("load latest Ethereum block: %w", err))
	}
	transactions := make([]RecentTransaction, 0, min(RecentTransactionLimit, len(block.Transactions)))
	for index := len(block.Transactions) - 1; index >= 0 && len(transactions) < RecentTransactionLimit; index-- {
		hash := block.Transactions[index]
		transactions = append(transactions, RecentTransaction{Hash: hash.Hex(), ExplorerURL: explorerURL(hash), Index: uint64(index)})
	}
	return RecentTransactions{BlockNumber: uint64(block.Number), Transactions: transactions}, nil
}

func (s *Service) Probe(ctx context.Context) (Readiness, error) {
	status := Readiness{}
	if s == nil || s.rpc == nil {
		return status, NewError(ErrorUnavailable, errors.New("transaction replay service is not configured"))
	}
	chainID, err := s.chainID(ctx)
	if err != nil {
		return status, err
	}
	status.RPC = true
	status.ChainID = chainID
	if chainID != ethereumMainnetChainID {
		return status, NewError(ErrorUnavailable, fmt.Errorf("configured RPC is chain %d; Ethereum Mainnet chain %d is required", chainID, ethereumMainnetChainID))
	}
	call := map[string]string{
		"from": "0x0000000000000000000000000000000000000000",
		"to":   "0x0000000000000000000000000000000000000000",
		"gas":  "0x5208", "gasPrice": "0x0", "value": "0x0", "data": "0x",
	}
	var prestate json.RawMessage
	prestateConfig := map[string]any{"tracer": "prestateTracer", "tracerConfig": map[string]any{"diffMode": false}}
	if err := s.rpc.CallContext(ctx, &prestate, "debug_traceCall", call, "latest", prestateConfig); err != nil {
		return status, NewError(ErrorUnavailable, fmt.Errorf("RPC cannot provide prestateTracer: %w", err))
	}
	if len(prestate) == 0 || string(prestate) == "null" || !json.Valid(prestate) {
		return status, NewError(ErrorUnavailable, errors.New("RPC returned an invalid prestateTracer capability response"))
	}
	status.PrestateTracer = true
	var opcodeTrace json.RawMessage
	traceConfig := map[string]any{"disableMemory": true, "disableStorage": true, "disableStack": false, "enableReturnData": true}
	if err := s.rpc.CallContext(ctx, &opcodeTrace, "debug_traceCall", call, "latest", traceConfig); err != nil {
		return status, NewError(ErrorUnavailable, fmt.Errorf("RPC cannot provide opcode traces: %w", err))
	}
	if len(opcodeTrace) == 0 || string(opcodeTrace) == "null" || !json.Valid(opcodeTrace) {
		return status, NewError(ErrorUnavailable, errors.New("RPC returned an invalid opcode trace capability response"))
	}
	status.OpcodeTrace = true
	status.Ready = true
	return status, nil
}

func (s *Service) transaction(ctx context.Context, hash common.Hash) (*types.Transaction, rawTransaction, error) {
	var raw json.RawMessage
	if err := s.rpc.CallContext(ctx, &raw, "eth_getTransactionByHash", hash); err != nil {
		return nil, rawTransaction{}, NewError(ErrorUpstream, fmt.Errorf("load transaction: %w", err))
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, rawTransaction{}, NewError(ErrorNotFound, fmt.Errorf("transaction %s was not found", hash.Hex()))
	}
	var tx types.Transaction
	if err := json.Unmarshal(raw, &tx); err != nil {
		return nil, rawTransaction{}, NewError(ErrorUpstream, fmt.Errorf("decode transaction: %w", err))
	}
	var fields struct {
		From             common.Address `json:"from"`
		BlockHash        common.Hash    `json:"blockHash"`
		BlockNumber      hexutil.Uint64 `json:"blockNumber"`
		TransactionIndex hexutil.Uint64 `json:"transactionIndex"`
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, rawTransaction{}, NewError(ErrorUpstream, fmt.Errorf("decode transaction metadata: %w", err))
	}
	if fields.BlockHash == (common.Hash{}) {
		return nil, rawTransaction{}, NewError(ErrorConflict, errors.New("pending transactions cannot be replayed"))
	}
	return &tx, rawTransaction{From: fields.From, BlockHash: fields.BlockHash, BlockNumber: uint64(fields.BlockNumber), Index: uint64(fields.TransactionIndex)}, nil
}

type prestateAccount struct {
	Balance *hexutil.Big                `json:"balance"`
	Nonce   flexibleUint64              `json:"nonce"`
	Code    hexutil.Bytes               `json:"code"`
	Storage map[common.Hash]common.Hash `json:"storage"`
}

type stateDiffAccount struct {
	Balance *hexutil.Big                `json:"balance"`
	Nonce   *flexibleUint64             `json:"nonce"`
	Code    *hexutil.Bytes              `json:"code"`
	Storage map[common.Hash]common.Hash `json:"storage"`
}

type transactionStateDiff struct {
	Pre  map[string]stateDiffAccount `json:"pre"`
	Post map[string]stateDiffAccount `json:"post"`
}

func (s *Service) prestate(ctx context.Context, hash common.Hash) (map[common.Address]prestateAccount, error) {
	var raw map[string]prestateAccount
	config := map[string]any{"tracer": "prestateTracer", "tracerConfig": map[string]any{"diffMode": false, "includeEmpty": true}}
	if err := s.rpc.CallContext(ctx, &raw, "debug_traceTransaction", hash, config); err != nil {
		return nil, NewError(ErrorUnavailable, fmt.Errorf("RPC cannot provide transaction prestate: %w", err))
	}
	state := make(map[common.Address]prestateAccount, len(raw))
	for address, account := range raw {
		if !common.IsHexAddress(address) {
			return nil, fmt.Errorf("prestate contains invalid address %q", address)
		}
		state[common.HexToAddress(address)] = account
	}
	if len(state) == 0 {
		return nil, NewError(ErrorUpstream, errors.New("RPC returned an empty transaction prestate"))
	}
	return state, nil
}

func (s *Service) stateDiff(ctx context.Context, hash common.Hash) (transactionStateDiff, error) {
	var diff transactionStateDiff
	config := map[string]any{"tracer": "prestateTracer", "tracerConfig": map[string]any{"diffMode": true}}
	if err := s.rpc.CallContext(ctx, &diff, "debug_traceTransaction", hash, config); err != nil {
		return transactionStateDiff{}, NewError(ErrorUnavailable, fmt.Errorf("RPC cannot provide transaction state diff: %w", err))
	}
	if diff.Pre == nil {
		diff.Pre = map[string]stateDiffAccount{}
	}
	if diff.Post == nil {
		diff.Post = map[string]stateDiffAccount{}
	}
	return diff, nil
}

type flexibleUint64 uint64

func (v *flexibleUint64) UnmarshalJSON(data []byte) error {
	text := strings.Trim(string(data), `"`)
	base := 10
	if strings.HasPrefix(text, "0x") {
		base, text = 16, strings.TrimPrefix(text, "0x")
	}
	parsed, err := strconv.ParseUint(text, base, 64)
	if err != nil {
		return err
	}
	*v = flexibleUint64(parsed)
	return nil
}

type rpcStructLog struct {
	PC      flexibleUint64  `json:"pc"`
	Op      json.RawMessage `json:"op"`
	OpName  string          `json:"opName"`
	Gas     flexibleUint64  `json:"gas"`
	GasCost flexibleUint64  `json:"gasCost"`
	Depth   int             `json:"depth"`
	Stack   []string        `json:"stack"`
	Error   string          `json:"error"`
}

type rpcExecutionTrace struct {
	Gas         flexibleUint64 `json:"gas"`
	Failed      bool           `json:"failed"`
	ReturnValue string         `json:"returnValue"`
	StructLogs  []rpcStructLog `json:"structLogs"`
}

func (s *Service) referenceTrace(ctx context.Context, hash common.Hash, receipt *types.Receipt) (differential.ExecutionResult, error) {
	var raw rpcExecutionTrace
	config := map[string]any{"disableMemory": true, "disableStorage": true, "disableStack": false, "enableReturnData": true, "limit": MaxTraceSteps + 1}
	if err := s.rpc.CallContext(ctx, &raw, "debug_traceTransaction", hash, config); err != nil {
		return differential.ExecutionResult{}, NewError(ErrorUnavailable, fmt.Errorf("RPC cannot trace transaction opcodes: %w", err))
	}
	if len(raw.StructLogs) > MaxTraceSteps {
		return differential.ExecutionResult{}, fmt.Errorf("reference trace has %d steps; maximum is %d", len(raw.StructLogs), MaxTraceSteps)
	}
	trace := make([]differential.NormalizedStep, 0, len(raw.StructLogs))
	for index, item := range raw.StructLogs {
		op, opName := decodeOpcode(item.Op, item.OpName)
		step := differential.NormalizedStep{
			Index: index, Depth: max(item.Depth-1, 0), PC: uint64(item.PC), Opcode: fmt.Sprintf("0x%02x", op), OpcodeName: opName,
			GasBefore: uint64(item.Gas), GasAfter: uint64(item.Gas) - min(uint64(item.Gas), uint64(item.GasCost)), StackBefore: canonicalStack(item.Stack),
		}
		trace = append(trace, step)
	}
	status := differential.StatusSuccess
	if receipt.Status == types.ReceiptStatusFailed || raw.Failed {
		status = differential.StatusFault
		if len(raw.StructLogs) > 0 {
			_, finalName := decodeOpcode(raw.StructLogs[len(raw.StructLogs)-1].Op, raw.StructLogs[len(raw.StructLogs)-1].OpName)
			if finalName == "REVERT" {
				status = differential.StatusRevert
			}
		}
	}
	return differential.ExecutionResult{Engine: "Geth RPC", EngineVersion: "network", Status: status, ReturnData: normalizeHex(raw.ReturnValue), GasUsed: receipt.GasUsed, Storage: map[string]string{}, Trace: trace}, nil
}

func decodeOpcode(raw json.RawMessage, name string) (byte, string) {
	if opcode, ok := opcodeByRPCName(name); ok {
		return opcode, core.OpcodeName(opcode)
	}
	var stringValue string
	if json.Unmarshal(raw, &stringValue) == nil {
		if opcode, ok := opcodeByRPCName(stringValue); ok {
			return opcode, core.OpcodeName(opcode)
		}
		if value, err := strconv.ParseUint(strings.TrimPrefix(strings.ToLower(stringValue), "0x"), 16, 8); err == nil {
			opcode := byte(value)
			return opcode, core.OpcodeName(opcode)
		}
	}
	var number byte
	if json.Unmarshal(raw, &number) == nil {
		return number, core.OpcodeName(number)
	}
	return 0, name
}

func opcodeByRPCName(name string) (byte, bool) {
	name = strings.ToUpper(strings.TrimSpace(name))
	if name == "KECCAK256" {
		return core.SHA3, true
	}
	return core.OpcodeByName(name)
}

func runEcho(ctx context.Context, tx *types.Transaction, sender common.Address, chainID uint64, header *types.Header, prestate map[common.Address]prestateAccount, collectEvidence bool, maxMemoryBytes int) (differential.ExecutionResult, *core.MemoryStateDB, []explaintrace.OpcodeEvent, error) {
	state := core.NewMemoryStateDB()
	for address, account := range prestate {
		state.CreateAccount(address)
		if account.Balance != nil {
			state.AddBalance(address, (*big.Int)(account.Balance))
		}
		state.SetNonce(address, uint64(account.Nonce))
		state.SetCode(address, account.Code)
		for key, value := range account.Storage {
			state.InitState(address, key, value)
		}
	}
	trace := make([]differential.NormalizedStep, 0, 1024)
	pending := make(map[int]int)
	overflow := false
	var collector *explaintrace.Collector
	if collectEvidence {
		collector = explaintrace.NewCollector(maxMemoryBytes)
	}
	hook := func(raw vm.TraceStep) bool {
		if ctx.Err() != nil {
			return false
		}
		if raw.IsPost {
			if index, ok := pending[raw.Depth]; ok && index < len(trace) {
				trace[index].GasAfter = raw.Gas
				if !raw.Halt {
					trace[index].StackAfter = canonicalStack(raw.Stack)
				}
			}
			if collector != nil {
				collector.Consume(raw)
			}
			return true
		}
		if len(trace) >= MaxTraceSteps {
			overflow = true
			return true
		}
		step := differential.NormalizedStep{Index: len(trace), Depth: raw.Depth, PC: raw.PC, Opcode: fmt.Sprintf("0x%02x", raw.Opcode), OpcodeName: raw.OpcodeName, GasBefore: raw.Gas, StackBefore: canonicalStack(raw.Stack), Address: raw.Address}
		trace = append(trace, step)
		pending[raw.Depth] = len(trace) - 1
		if collector != nil {
			collector.Consume(raw)
		}
		return true
	}
	ctxBlock := &vm.BlockContext{BlockNumber: header.Number, Timestamp: header.Time, Coinbase: header.Coinbase, GasLimit: header.GasLimit, BaseFee: header.BaseFee, Difficulty: header.Difficulty, Random: new(big.Int).SetBytes(header.MixDigest[:]), ChainID: new(big.Int).SetUint64(chainID)}
	if header.ExcessBlobGas != nil {
		ctxBlock.BlobBaseFee = eip4844.CalcBlobFee(params.MainnetChainConfig, header)
	}
	var output []byte
	var gasUsed uint64
	var reverted bool
	var executionErr error
	if collector != nil {
		output, gasUsed, reverted, executionErr = vm.ApplyTransactionWithContextAndDetailedHook(state, tx, sender, ctxBlock, hook)
	} else {
		output, gasUsed, reverted, executionErr = vm.ApplyTransactionWithContextAndHook(state, tx, sender, ctxBlock, hook)
	}
	if ctx.Err() != nil {
		return differential.ExecutionResult{}, nil, nil, ctx.Err()
	}
	if overflow {
		return differential.ExecutionResult{}, nil, nil, fmt.Errorf("EchoEVM trace exceeds maximum %d steps", MaxTraceSteps)
	}
	status := differential.StatusSuccess
	if executionErr != nil {
		status = differential.StatusFault
	} else if reverted {
		status = differential.StatusRevert
	}
	result := differential.ExecutionResult{Engine: "EchoEVM", EngineVersion: echoModuleVersion(), Status: status, ReturnData: "0x" + hex.EncodeToString(output), GasUsed: gasUsed, Storage: map[string]string{}, Trace: trace}
	if executionErr != nil {
		result.Error = executionErr.Error()
	}
	var evidenceEvents []explaintrace.OpcodeEvent
	if collector != nil {
		evidenceEvents = collector.Events()
	}
	return result, state, evidenceEvents, nil
}

func echoModuleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Path != "github.com/smallyunet/echoevm" || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "devel"
	}
	return info.Main.Version
}

func compareState(state *core.MemoryStateDB, diff transactionStateDiff) (map[string]string, map[string]string) {
	echo, geth := make(map[string]string), make(map[string]string)
	for rawAddress, account := range diff.Post {
		address := common.HexToAddress(rawAddress)
		prefix := strings.ToLower(address.Hex())
		if account.Balance != nil {
			key := prefix + ":balance"
			echo[key], geth[key] = state.GetBalance(address).String(), (*big.Int)(account.Balance).String()
		}
		if account.Nonce != nil {
			key := prefix + ":nonce"
			echo[key], geth[key] = strconv.FormatUint(state.GetNonce(address), 10), strconv.FormatUint(uint64(*account.Nonce), 10)
		}
		if account.Code != nil {
			key := prefix + ":code"
			echo[key], geth[key] = hexutil.Encode(state.GetCode(address)), hexutil.Encode(*account.Code)
		}
		for slot, value := range account.Storage {
			key := prefix + ":storage:" + slot.Hex()
			echo[key], geth[key] = state.GetState(address, slot).Hex(), value.Hex()
		}
	}
	for rawAddress, account := range diff.Pre {
		address := common.HexToAddress(rawAddress)
		prefix := strings.ToLower(address.Hex())
		if _, exists := diff.Post[rawAddress]; !exists {
			key := prefix + ":deleted"
			echo[key], geth[key] = strconv.FormatBool(state.HasSuicided(address)), "true"
			continue
		}
		post := diff.Post[rawAddress]
		for slot := range account.Storage {
			if _, exists := post.Storage[slot]; exists {
				continue
			}
			key := prefix + ":storage:" + slot.Hex()
			echo[key], geth[key] = state.GetState(address, slot).Hex(), (common.Hash{}).Hex()
		}
	}
	return echo, geth
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func summarize(tx *types.Transaction, meta rawTransaction, receipt *types.Receipt, header *types.Header, chainID uint64) TransactionSummary {
	var to *string
	if tx.To() != nil {
		value := tx.To().Hex()
		to = &value
	}
	status := "success"
	if receipt.Status == types.ReceiptStatusFailed {
		status = "reverted"
	}
	return TransactionSummary{Hash: tx.Hash().Hex(), ExplorerURL: explorerURL(tx.Hash()), ChainID: chainID, BlockNumber: meta.BlockNumber, BlockHash: meta.BlockHash.Hex(), Index: meta.Index, From: meta.From.Hex(), To: to, Value: tx.Value().String(), GasLimit: tx.Gas(), GasUsed: receipt.GasUsed, Type: tx.Type(), Input: hexutil.Encode(tx.Data()), Status: status, Fork: forkName(header.Time)}
}

func explorerURL(hash common.Hash) string {
	return "https://etherscan.io/tx/" + hash.Hex()
}

func forkName(timestamp uint64) string {
	cancun, prague, osaka := uint64(1710338135), uint64(1746612311), uint64(1764798551)
	switch {
	case timestamp >= osaka:
		return "Osaka"
	case timestamp >= prague:
		return "Prague"
	case timestamp >= cancun:
		return "Cancun"
	default:
		return "Pre-Cancun"
	}
}

func replayWarnings(timestamp uint64, tx *types.Transaction, echo differential.ExecutionResult) []string {
	warnings := make([]string, 0, 3)
	if forkName(timestamp) != "Cancun" {
		warnings = append(warnings, "EchoEVM currently executes Cancun rules; this transaction belongs to "+forkName(timestamp)+", so a divergence may reflect unsupported fork semantics.")
	}
	if tx.Type() == types.SetCodeTxType {
		warnings = append(warnings, "EIP-7702 set-code transaction semantics are not implemented by EchoEVM.")
	}
	for _, step := range echo.Trace {
		if step.OpcodeName == "BLOCKHASH" {
			warnings = append(warnings, "BLOCKHASH currently resolves to zero because historical block hashes are not part of the replay witness.")
			break
		}
	}
	return warnings
}

func canonicalStack(values []string) []string {
	out := make([]string, len(values))
	for index, value := range values {
		value = strings.TrimPrefix(strings.ToLower(value), "0x")
		value = strings.TrimLeft(value, "0")
		if value == "" {
			value = "0"
		}
		out[index] = "0x" + value
	}
	return out
}

func normalizeHex(value string) string {
	value = strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if value == "" {
		return "0x"
	}
	return "0x" + strings.ToLower(value)
}

func compare(echo, geth differential.ExecutionResult) Result {
	result := Result{StatusMatch: echo.Status == geth.Status, ReturnDataMatch: echo.ReturnData == geth.ReturnData, GasMatch: echo.GasUsed == geth.GasUsed, StateMatch: true, EchoEVM: echo, Geth: geth, EchoState: map[string]string{}, GethState: map[string]string{}}
	limit := min(len(echo.Trace), len(geth.Trace))
	result.TraceMatch = len(echo.Trace) == len(geth.Trace)
	for index := 0; index < limit; index++ {
		a, b := echo.Trace[index], geth.Trace[index]
		if a.Depth != b.Depth || a.PC != b.PC || a.Opcode != b.Opcode {
			step, pc := index, a.PC
			result.FirstDivergence = &differential.Divergence{Kind: "trace", Step: &step, PC: &pc, Opcode: a.OpcodeName, Field: "instruction", EchoEVM: fmt.Sprintf("depth=%d pc=%d %s", a.Depth, a.PC, a.OpcodeName), Geth: fmt.Sprintf("depth=%d pc=%d %s", b.Depth, b.PC, b.OpcodeName), Description: "transaction instruction stream diverged"}
			result.TraceMatch = false
			break
		}
		aCost, bCost := traceGasCost(a), traceGasCost(b)
		if comparableTraceGasCost(a.OpcodeName) && aCost != bCost {
			step, pc := index, a.PC
			result.FirstDivergence = &differential.Divergence{Kind: "trace", Step: &step, PC: &pc, Opcode: a.OpcodeName, Field: "gasCost", EchoEVM: aCost, Geth: bCost, Description: "transaction opcode gas cost diverged"}
			result.TraceMatch = false
			break
		}
	}
	if result.FirstDivergence == nil && len(echo.Trace) != len(geth.Trace) {
		step := limit
		result.FirstDivergence = &differential.Divergence{Kind: "trace", Step: &step, Field: "length", EchoEVM: len(echo.Trace), Geth: len(geth.Trace), Description: "transaction trace lengths differ"}
	}
	if result.FirstDivergence == nil {
		switch {
		case !result.StatusMatch:
			result.FirstDivergence = &differential.Divergence{Kind: "result", Field: "status", EchoEVM: echo.Status, Geth: geth.Status}
		case !result.ReturnDataMatch:
			result.FirstDivergence = &differential.Divergence{Kind: "result", Field: "returnData", EchoEVM: echo.ReturnData, Geth: geth.ReturnData}
		case !result.GasMatch:
			result.FirstDivergence = &differential.Divergence{Kind: "result", Field: "gasUsed", EchoEVM: echo.GasUsed, Geth: geth.GasUsed}
		}
	}
	result.Match = result.StatusMatch && result.ReturnDataMatch && result.GasMatch && result.TraceMatch
	return result
}

func traceGasCost(step differential.NormalizedStep) uint64 {
	if step.GasAfter >= step.GasBefore {
		return 0
	}
	return step.GasBefore - step.GasAfter
}

func comparableTraceGasCost(opcode string) bool {
	switch opcode {
	case "CALL", "CALLCODE", "DELEGATECALL", "STATICCALL", "CREATE", "CREATE2":
		return false
	default:
		return true
	}
}
