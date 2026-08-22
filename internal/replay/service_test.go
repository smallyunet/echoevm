package replay

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/smallyunet/echoevm/internal/differential"
	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/crypto"
	"github.com/smallyunet/echoevm/internal/eth/hexutil"
	"github.com/smallyunet/echoevm/internal/eth/types"
	explaintrace "github.com/smallyunet/echoevm/internal/trace"
)

type replayFixtureRPC struct {
	chainID     uint64
	methodCalls map[string]int
	raw         json.RawMessage
	header      types.Header
	receipt     types.Receipt
	prestate    map[string]prestateAccount
	diff        transactionStateDiff
	reference   rpcExecutionTrace
	recentBlock rpcRecentBlock
}

func (f *replayFixtureRPC) CallContext(_ context.Context, result any, method string, args ...any) error {
	if f.methodCalls == nil {
		f.methodCalls = make(map[string]int)
	}
	f.methodCalls[method]++
	switch method {
	case "eth_chainId":
		chainID := f.chainID
		if chainID == 0 {
			chainID = ethereumMainnetChainID
		}
		*result.(*hexutil.Uint64) = hexutil.Uint64(chainID)
	case "eth_getTransactionByHash":
		*result.(*json.RawMessage) = append(json.RawMessage(nil), f.raw...)
	case "eth_getTransactionReceipt":
		*result.(*types.Receipt) = f.receipt
	case "eth_getBlockByHash":
		*result.(*types.Header) = f.header
	case "eth_getBlockByNumber":
		*result.(*rpcRecentBlock) = f.recentBlock
	case "debug_traceTransaction":
		config := args[1].(map[string]any)
		if config["tracer"] == "prestateTracer" {
			tracerConfig := config["tracerConfig"].(map[string]any)
			if tracerConfig["diffMode"] == true {
				*result.(*transactionStateDiff) = f.diff
			} else {
				*result.(*map[string]prestateAccount) = f.prestate
			}
		} else {
			*result.(*rpcExecutionTrace) = f.reference
		}
	case "debug_traceCall":
		*result.(*json.RawMessage) = json.RawMessage(`{}`)
	default:
		panic("unexpected RPC method " + method)
	}
	return nil
}

func TestRecentTransactionsLoadsLatestMainnetBlock(t *testing.T) {
	hashes := []common.Hash{
		common.HexToHash("0x01"), common.HexToHash("0x02"), common.HexToHash("0x03"),
		common.HexToHash("0x04"), common.HexToHash("0x05"), common.HexToHash("0x06"),
	}
	rpc := &replayFixtureRPC{recentBlock: rpcRecentBlock{Number: hexutil.Uint64(123), Transactions: hashes}}
	result, err := NewVerificationServiceWithCaller(rpc).RecentTransactions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.BlockNumber != 123 || len(result.Transactions) != RecentTransactionLimit {
		t.Fatalf("recent transactions = %+v", result)
	}
	if result.Transactions[0].Hash != hashes[5].Hex() || result.Transactions[0].Index != 5 || result.Transactions[4].Hash != hashes[1].Hex() {
		t.Fatalf("unexpected transaction order: %+v", result.Transactions)
	}
	if rpc.methodCalls["eth_chainId"] != 1 || rpc.methodCalls["eth_getBlockByNumber"] != 1 {
		t.Fatalf("RPC calls = %+v", rpc.methodCalls)
	}
}

func TestProbeRequiresMainnetTraceCapabilities(t *testing.T) {
	status, err := NewVerificationServiceWithCaller(&replayFixtureRPC{}).Probe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status.Ready || status.ChainID != ethereumMainnetChainID || !status.RPC || !status.PrestateTracer || !status.OpcodeTrace {
		t.Fatalf("readiness = %+v", status)
	}
}

func TestReplayRejectsNonMainnetRPC(t *testing.T) {
	_, err := NewVerificationServiceWithCaller(&replayFixtureRPC{chainID: 11155111}).Verify(context.Background(), VerificationRequest{Input: testHash})
	if err == nil || err.Error() != "configured RPC is chain 11155111; Ethereum Mainnet chain 1 is required" {
		t.Fatalf("Replay error = %v", err)
	}
	if ErrorKindOf(err) != ErrorUnavailable {
		t.Fatalf("error kind = %q", ErrorKindOf(err))
	}
}

func TestReplayHydratesPrestateAndExecutesTransaction(t *testing.T) {
	key, err := crypto.HexToECDSA("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	sender := crypto.PubkeyToAddress(key.PublicKey)
	recipient := common.HexToAddress("0x2000000000000000000000000000000000000002")
	tx, err := types.SignTx(types.NewTransaction(0, recipient, big.NewInt(0), 21_000, big.NewInt(1), nil), types.NewEIP155Signer(big.NewInt(1)), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tx.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	blockHash := common.HexToHash("0x1234")
	coinbase := common.HexToAddress("0x3000000000000000000000000000000000000003")
	postNonce := flexibleUint64(1)
	fields["from"] = sender.Hex()
	fields["blockHash"] = blockHash.Hex()
	fields["blockNumber"] = "0x1"
	fields["transactionIndex"] = "0x0"
	raw, _ = json.Marshal(fields)
	fixture := &replayFixtureRPC{
		raw:     raw,
		header:  types.Header{Number: big.NewInt(1), Time: 1710338135, GasLimit: 30_000_000, Difficulty: new(big.Int), Coinbase: coinbase},
		receipt: types.Receipt{TxHash: tx.Hash(), Status: types.ReceiptStatusSuccessful, GasUsed: 21_000},
		prestate: map[string]prestateAccount{
			sender.Hex():    {Balance: (*hexutil.Big)(big.NewInt(1_000_000)), Nonce: 0},
			recipient.Hex(): {Balance: (*hexutil.Big)(new(big.Int)), Nonce: 0},
		},
		reference: rpcExecutionTrace{Gas: 21_000, ReturnValue: "", StructLogs: nil},
		diff: transactionStateDiff{
			Pre: map[string]stateDiffAccount{sender.Hex(): {}},
			Post: map[string]stateDiffAccount{
				sender.Hex():   {Balance: (*hexutil.Big)(big.NewInt(979_000)), Nonce: &postNonce},
				coinbase.Hex(): {Balance: (*hexutil.Big)(big.NewInt(21_000))},
			},
		},
	}
	result, err := NewVerificationServiceWithCaller(fixture).Verify(context.Background(), VerificationRequest{
		Input: tx.Hash().Hex(), Profile: explaintrace.ProfileAuto, Limit: DefaultEvidenceLimit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Match || result.Transaction.Hash != tx.Hash().Hex() {
		t.Fatalf("result match=%t hash=%s", result.Match, result.Transaction.Hash)
	}
	if result.EchoEVM.GasUsed != 21_000 || len(result.EchoEVM.Trace) != 0 {
		t.Fatalf("EchoEVM gas=%d trace=%d", result.EchoEVM.GasUsed, len(result.EchoEVM.Trace))
	}
	if result.Evidence == nil || result.Evidence.Schema != explaintrace.EvidenceSchemaVersion || result.Evidence.Transaction.Hash != tx.Hash().Hex() || !result.Evidence.Comparison.Match {
		t.Fatalf("replay evidence = %+v", result.Evidence)
	}
}

func TestImportDebugWitnessDoesNotRequestReferenceExecution(t *testing.T) {
	key, err := crypto.HexToECDSA("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	sender := crypto.PubkeyToAddress(key.PublicKey)
	recipient := common.HexToAddress("0x2000000000000000000000000000000000000002")
	tx, err := types.SignTx(types.NewTransaction(0, recipient, new(big.Int), 21_000, big.NewInt(1), nil), types.NewEIP155Signer(big.NewInt(1)), key)
	if err != nil {
		t.Fatal(err)
	}
	header := types.Header{Number: big.NewInt(1), Time: 1710338135, GasLimit: 30_000_000, Difficulty: new(big.Int)}
	raw, err := transactionRPCJSON(tx, sender, header.Hash(), 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	fixture := &replayFixtureRPC{
		raw:      raw,
		header:   header,
		prestate: map[string]prestateAccount{sender.Hex(): {Balance: (*hexutil.Big)(big.NewInt(1_000_000))}},
	}
	witness, err := NewVerificationServiceWithCaller(fixture).ImportDebugWitness(context.Background(), tx.Hash().Hex())
	if err != nil {
		t.Fatal(err)
	}
	if witness.Schema != WitnessSchemaVersion || len(witness.Transaction) == 0 || len(witness.Prestate) != 1 {
		t.Fatalf("witness = %+v", witness)
	}
	if fixture.methodCalls["debug_traceTransaction"] != 1 {
		t.Fatalf("debug trace calls = %d, want one prestate import", fixture.methodCalls["debug_traceTransaction"])
	}
}

func transactionRPCJSON(tx *types.Transaction, sender common.Address, blockHash common.Hash, blockNumber, index uint64) (json.RawMessage, error) {
	raw, err := tx.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	fields["from"] = sender.Hex()
	fields["blockHash"] = blockHash.Hex()
	fields["blockNumber"] = hexutil.EncodeUint64(blockNumber)
	fields["transactionIndex"] = hexutil.EncodeUint64(index)
	return json.Marshal(fields)
}

func TestRunEchoCollectsCausalEvidenceForRevertedStorage(t *testing.T) {
	sender := common.HexToAddress("0x1000000000000000000000000000000000000001")
	recipient := common.HexToAddress("0x2000000000000000000000000000000000000002")
	tx := types.NewTransaction(0, recipient, big.NewInt(0), 100_000, big.NewInt(1), nil)
	prestate := map[common.Address]prestateAccount{
		sender:    {Balance: (*hexutil.Big)(big.NewInt(1_000_000)), Nonce: 0},
		recipient: {Code: hexutil.Bytes{0x60, 0x01, 0x60, 0x00, 0x55, 0x60, 0x00, 0x60, 0x00, 0xfd}},
	}
	header := &types.Header{Number: big.NewInt(1), Time: 1710338135, GasLimit: 30_000_000, Difficulty: new(big.Int)}

	result, _, events, err := runEcho(context.Background(), tx, sender, ethereumMainnetChainID, header, prestate, nil, true, DefaultEvidenceMemoryBytes)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != differential.StatusRevert || len(events) == 0 {
		t.Fatalf("status=%s events=%d", result.Status, len(events))
	}
	document, err := explaintrace.BuildEvidence(explaintrace.ExecutionResult{
		Status: string(result.Status), GasLimit: tx.Gas(), GasUsed: result.GasUsed, ReturnData: result.ReturnData, TotalSteps: len(events),
	}, events, explaintrace.ProfileStorage, DefaultEvidenceLimit)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Events) != 2 || document.Events[0].Op != "SSTORE" || document.Events[1].Op != "REVERT" {
		t.Fatalf("events = %+v", document.Events)
	}
	if len(document.Links) != 1 || document.Links[0].Kind != "rolls-back" {
		t.Fatalf("links = %+v", document.Links)
	}
}

func TestReplayRejectsInvalidEvidenceOptionsBeforeRPC(t *testing.T) {
	rpc := &replayFixtureRPC{}
	_, err := NewVerificationServiceWithCaller(rpc).Verify(context.Background(), VerificationRequest{Input: testHash, Profile: "mystery"})
	if err == nil || !strings.Contains(err.Error(), "unsupported evidence profile") {
		t.Fatalf("Replay error = %v", err)
	}
	if len(rpc.methodCalls) != 0 {
		t.Fatalf("RPC calls = %+v", rpc.methodCalls)
	}
}

func TestDecodeOpcodeCanonicalizesRPCForms(t *testing.T) {
	tests := []struct {
		name   string
		raw    json.RawMessage
		opName string
	}{
		{name: "geth mnemonic", raw: json.RawMessage(`"KECCAK256"`)},
		{name: "legacy mnemonic field", raw: json.RawMessage(`null`), opName: "KECCAK256"},
		{name: "hex string", raw: json.RawMessage(`"0x20"`)},
		{name: "numeric opcode", raw: json.RawMessage(`32`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opcode, name := decodeOpcode(test.raw, test.opName)
			if opcode != 0x20 || name != "SHA3" {
				t.Fatalf("decodeOpcode(%s, %q) = %#x %q", test.raw, test.opName, opcode, name)
			}
		})
	}
}

func TestReplayWarningsDetectPartialBlockHashWitness(t *testing.T) {
	execution := differential.ExecutionResult{Trace: []differential.NormalizedStep{
		{OpcodeName: "BLOCKHASH", StackBefore: []string{"0x63"}},
	}}

	if warnings := replayWarnings(1710338135, 100, execution, map[uint64]common.Hash{98: common.HexToHash("0x01")}); len(warnings) != 1 {
		t.Fatalf("warnings = %v", warnings)
	}
	if warnings := replayWarnings(1710338135, 100, execution, map[uint64]common.Hash{99: common.HexToHash("0x01")}); len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestCompareReportsFirstOpcodeGasCostDivergence(t *testing.T) {
	echo := differential.ExecutionResult{Status: differential.StatusSuccess, ReturnData: "0x", GasUsed: 3_000, Trace: []differential.NormalizedStep{{Index: 0, PC: 5, Opcode: "0x55", OpcodeName: "SSTORE", GasBefore: 10_000, GasAfter: 7_000}}}
	geth := differential.ExecutionResult{Status: differential.StatusSuccess, ReturnData: "0x", GasUsed: 2_900, Trace: []differential.NormalizedStep{{Index: 0, PC: 5, Opcode: "0x55", OpcodeName: "SSTORE", GasBefore: 10_000, GasAfter: 7_100}}}

	result := compare(echo, geth)
	if result.TraceMatch || result.FirstDivergence == nil {
		t.Fatalf("comparison = %+v", result)
	}
	if result.FirstDivergence.Field != "gasCost" || result.FirstDivergence.EchoEVM != uint64(3_000) || result.FirstDivergence.Geth != uint64(2_900) {
		t.Fatalf("first divergence = %+v", result.FirstDivergence)
	}
}

func TestCompareSkipsNonComparableNestedCallGasDelta(t *testing.T) {
	echo := differential.ExecutionResult{Status: differential.StatusSuccess, ReturnData: "0x", GasUsed: 1_000, Trace: []differential.NormalizedStep{{Index: 0, PC: 5, Opcode: "0xf4", OpcodeName: "DELEGATECALL", GasBefore: 100_000, GasAfter: 20_000}}}
	geth := differential.ExecutionResult{Status: differential.StatusSuccess, ReturnData: "0x", GasUsed: 1_000, Trace: []differential.NormalizedStep{{Index: 0, PC: 5, Opcode: "0xf4", OpcodeName: "DELEGATECALL", GasBefore: 100_000, GasAfter: 50_000}}}

	result := compare(echo, geth)
	if !result.Match || !result.TraceMatch || result.FirstDivergence != nil {
		t.Fatalf("comparison = %+v", result)
	}
}
