package main

import "github.com/smallyunet/echoevm/internal/differential"

const agentSummarySchemaVersion = 1

type executionSummary struct {
	Engine        string              `json:"engine"`
	EngineVersion string              `json:"engineVersion"`
	Status        differential.Status `json:"status"`
	ReturnData    string              `json:"returnData"`
	GasUsed       uint64              `json:"gasUsed"`
	Storage       map[string]string   `json:"storage"`
	TraceSteps    int                 `json:"traceSteps"`
	Error         string              `json:"error,omitempty"`
}

type requestSummary struct {
	Fork                string `json:"fork"`
	GasLimit            uint64 `json:"gasLimit"`
	DeployGasLimit      uint64 `json:"deployGasLimit,omitempty"`
	BytecodeBytes       int    `json:"bytecodeBytes"`
	InitCodeBytes       int    `json:"initCodeBytes,omitempty"`
	CalldataBytes       int    `json:"calldataBytes"`
	InitialStorageSlots int    `json:"initialStorageSlots"`
}

func summarizeExecution(result differential.ExecutionResult) executionSummary {
	return executionSummary{
		Engine: result.Engine, EngineVersion: result.EngineVersion, Status: result.Status,
		ReturnData: result.ReturnData, GasUsed: result.GasUsed, Storage: result.Storage,
		TraceSteps: len(result.Trace), Error: result.Error,
	}
}

func summarizeRequest(req differential.Request) requestSummary {
	return requestSummary{
		Fork: req.Fork, GasLimit: req.GasLimit, DeployGasLimit: req.DeployGasLimit,
		BytecodeBytes: hexByteLength(req.Bytecode), InitCodeBytes: hexByteLength(req.InitCode),
		CalldataBytes: hexByteLength(req.Calldata), InitialStorageSlots: len(req.InitialStorage),
	}
}

func hexByteLength(value string) int {
	if len(value) >= 2 && value[:2] == "0x" {
		value = value[2:]
	}
	return len(value) / 2
}
