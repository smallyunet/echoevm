// Package replay turns an Ethereum transaction hash into a reproducible
// EchoEVM execution using transaction prestate supplied by a trace-capable RPC.
package replay

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/differential"
	explaintrace "github.com/smallyunet/echoevm/internal/trace"
)

const MaxTraceSteps = 50_000
const RecentTransactionLimit = 5
const DefaultEvidenceLimit = 40
const DefaultEvidenceMemoryBytes = 256

type Request struct {
	Input          string `json:"input"`
	Profile        string `json:"profile,omitempty"`
	Limit          int    `json:"limit,omitempty"`
	MaxMemoryBytes int    `json:"maxMemoryBytes,omitempty"`
}

type RecentTransaction struct {
	Hash        string `json:"hash"`
	ExplorerURL string `json:"explorerUrl"`
	Index       uint64 `json:"transactionIndex"`
}

type RecentTransactions struct {
	BlockNumber  uint64              `json:"blockNumber"`
	Transactions []RecentTransaction `json:"transactions"`
}

type TransactionSummary struct {
	Hash        string  `json:"hash"`
	ExplorerURL string  `json:"explorerUrl"`
	ChainID     uint64  `json:"chainId"`
	BlockNumber uint64  `json:"blockNumber"`
	BlockHash   string  `json:"blockHash"`
	Index       uint64  `json:"transactionIndex"`
	From        string  `json:"from"`
	To          *string `json:"to,omitempty"`
	Value       string  `json:"value"`
	GasLimit    uint64  `json:"gasLimit"`
	GasUsed     uint64  `json:"gasUsed"`
	Type        uint8   `json:"type"`
	Input       string  `json:"input"`
	Status      string  `json:"status"`
	Fork        string  `json:"fork"`
}

type Result struct {
	Match           bool                         `json:"match"`
	StatusMatch     bool                         `json:"statusMatch"`
	ReturnDataMatch bool                         `json:"returnDataMatch"`
	GasMatch        bool                         `json:"gasMatch"`
	StateMatch      bool                         `json:"stateMatch"`
	TraceMatch      bool                         `json:"traceMatch"`
	FirstDivergence *differential.Divergence     `json:"firstDivergence,omitempty"`
	Transaction     TransactionSummary           `json:"transaction"`
	EchoEVM         differential.ExecutionResult `json:"echoevm"`
	Geth            differential.ExecutionResult `json:"geth"`
	Warnings        []string                     `json:"warnings,omitempty"`
	EchoState       map[string]string            `json:"echoState"`
	GethState       map[string]string            `json:"gethState"`
	TraceSemantics  string                       `json:"traceSemantics"`
	Evidence        *EvidenceResult              `json:"evidence,omitempty"`
}

// EvidenceResult is the compact, replay-specific envelope presented to coding
// agents. It retains transaction/fork provenance and comparison confidence
// without duplicating both engines' complete opcode traces.
type EvidenceResult struct {
	explaintrace.EvidenceDocument
	Transaction TransactionSummary `json:"transaction"`
	Comparison  EvidenceComparison `json:"comparison"`
	Warnings    []string           `json:"warnings,omitempty"`
}

type EvidenceComparison struct {
	Match           bool                     `json:"match"`
	StatusMatch     bool                     `json:"statusMatch"`
	ReturnDataMatch bool                     `json:"returnDataMatch"`
	GasMatch        bool                     `json:"gasMatch"`
	StateMatch      bool                     `json:"stateMatch"`
	TraceMatch      bool                     `json:"traceMatch"`
	FirstDivergence *differential.Divergence `json:"firstDivergence,omitempty"`
}

type Readiness struct {
	Ready          bool   `json:"ready"`
	ChainID        uint64 `json:"chainId,omitempty"`
	RPC            bool   `json:"rpc"`
	PrestateTracer bool   `json:"prestateTracer"`
	OpcodeTrace    bool   `json:"opcodeTrace"`
}

type Caller interface {
	CallContext(context.Context, any, string, ...any) error
}

type transactionReference struct {
	Hash    common.Hash
	ChainID uint64
}

type rawTransaction struct {
	From        common.Address
	BlockHash   common.Hash
	BlockNumber uint64
	Index       uint64
}
