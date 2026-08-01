package replay

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smallyunet/echoevm/internal/differential"
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
	result, err := NewServiceWithCaller(rpc).RecentTransactions(context.Background())
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
	status, err := NewServiceWithCaller(&replayFixtureRPC{}).Probe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status.Ready || status.ChainID != ethereumMainnetChainID || !status.RPC || !status.PrestateTracer || !status.OpcodeTrace {
		t.Fatalf("readiness = %+v", status)
	}
}

func TestReplayRejectsNonMainnetRPC(t *testing.T) {
	_, err := NewServiceWithCaller(&replayFixtureRPC{chainID: 11155111}).Replay(context.Background(), Request{Input: testHash})
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
	result, err := NewServiceWithCaller(fixture).Replay(context.Background(), Request{Input: tx.Hash().Hex()})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Match || result.Transaction.Hash != tx.Hash().Hex() {
		t.Fatalf("result match=%t hash=%s", result.Match, result.Transaction.Hash)
	}
	if result.EchoEVM.GasUsed != 21_000 || len(result.EchoEVM.Trace) != 0 {
		t.Fatalf("EchoEVM gas=%d trace=%d", result.EchoEVM.GasUsed, len(result.EchoEVM.Trace))
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
