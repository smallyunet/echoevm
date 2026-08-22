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

func summarizeExecution(result differential.ExecutionResult) executionSummary {
	return executionSummary{
		Engine: result.Engine, EngineVersion: result.EngineVersion, Status: result.Status,
		ReturnData: result.ReturnData, GasUsed: result.GasUsed, Storage: result.Storage,
		TraceSteps: len(result.Trace), Error: result.Error,
	}
}
