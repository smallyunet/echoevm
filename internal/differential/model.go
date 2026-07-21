// Package differential compares an isolated EchoEVM execution with an embedded
// go-ethereum execution. It intentionally supports one explicit environment:
// Cancun rules, in-memory state, and no external RPC access.
package differential

import "context"

const (
	ForkCancun       = "Cancun"
	DefaultGasLimit  = uint64(1_000_000)
	MaxGasLimit      = uint64(30_000_000)
	MaxBytecodeBytes = 24_576
	MaxCalldataBytes = 128 * 1024
	MaxTraceSteps    = 2_000
)

type Request struct {
	Fork           string            `json:"fork"`
	Bytecode       string            `json:"bytecode"`
	Calldata       string            `json:"calldata"`
	GasLimit       uint64            `json:"gasLimit"`
	InitialStorage map[string]string `json:"initialStorage,omitempty"`
}

type Status string

const (
	StatusSuccess Status = "success"
	StatusRevert  Status = "revert"
	StatusFault   Status = "fault"
)

// NormalizedStep has pre-op identity and state, plus post-op state when it can
// be derived reliably on both engines. StackAfter is deliberately omitted for
// a terminal step because Geth's exit hook does not expose it.
type NormalizedStep struct {
	Index       int      `json:"index"`
	Depth       int      `json:"depth"`
	PC          uint64   `json:"pc"`
	Opcode      string   `json:"opcode"`
	OpcodeName  string   `json:"opcodeName"`
	GasBefore   uint64   `json:"gasBefore"`
	GasAfter    uint64   `json:"gasAfter"`
	StackBefore []string `json:"stackBefore"`
	StackAfter  []string `json:"stackAfter,omitempty"`
	HaltClass   Status   `json:"haltClass,omitempty"`
}

type ExecutionResult struct {
	Engine        string            `json:"engine"`
	EngineVersion string            `json:"engineVersion"`
	Status        Status            `json:"status"`
	ReturnData    string            `json:"returnData"`
	GasUsed       uint64            `json:"gasUsed"`
	Storage       map[string]string `json:"storage"`
	Trace         []NormalizedStep  `json:"trace"`
	Error         string            `json:"error,omitempty"`
}

type Divergence struct {
	Kind        string  `json:"kind"`
	Step        *int    `json:"step,omitempty"`
	PC          *uint64 `json:"pc,omitempty"`
	Opcode      string  `json:"opcode,omitempty"`
	Field       string  `json:"field"`
	EchoEVM     any     `json:"echoevm"`
	Geth        any     `json:"geth"`
	Description string  `json:"description"`
}

type ComparisonResult struct {
	Match           bool            `json:"match"`
	StatusMatch     bool            `json:"statusMatch"`
	ReturnDataMatch bool            `json:"returnDataMatch"`
	GasMatch        bool            `json:"gasMatch"`
	StorageMatch    bool            `json:"storageMatch"`
	TraceMatch      bool            `json:"traceMatch"`
	FirstDivergence *Divergence     `json:"firstDivergence,omitempty"`
	EchoEVM         ExecutionResult `json:"echoevm"`
	Geth            ExecutionResult `json:"geth"`
	Request         Request         `json:"request"`
	TraceSemantics  string          `json:"traceSemantics"`
}

type Runner interface {
	Run(context.Context, Request) (ExecutionResult, error)
}
