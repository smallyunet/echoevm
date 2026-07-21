// Package replay turns an Ethereum transaction hash into a reproducible
// EchoEVM execution using transaction prestate supplied by a trace-capable RPC.
package replay

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/differential"
)

const MaxTraceSteps = 50_000

type Request struct {
	Input string `json:"input"`
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
