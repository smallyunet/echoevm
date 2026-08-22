//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/smallyunet/echoevm/internal/replay"
)

var replayFunction js.Func

type browserOptions struct {
	Profile        string `json:"profile"`
	Limit          int    `json:"limit"`
	MaxMemoryBytes int    `json:"maxMemoryBytes"`
}

type browserExecution struct {
	Engine        string `json:"engine"`
	EngineVersion string `json:"engineVersion"`
	Status        string `json:"status"`
	ReturnData    string `json:"returnData"`
	GasUsed       uint64 `json:"gasUsed"`
	TotalSteps    int    `json:"totalSteps"`
	StateEntries  int    `json:"stateEntries"`
	Error         string `json:"error,omitempty"`
}

type browserReplayResult struct {
	Transaction replay.TransactionSummary    `json:"transaction"`
	Execution   browserExecution             `json:"execution"`
	Warnings    []string                     `json:"warnings,omitempty"`
	Witness     replay.WitnessProvenance     `json:"witness"`
	Evidence    *replay.ReplayEvidenceResult `json:"evidence,omitempty"`
}

type browserResponse struct {
	OK     bool                 `json:"ok"`
	Result *browserReplayResult `json:"result,omitempty"`
	Error  string               `json:"error,omitempty"`
}

func main() {
	replayFunction = js.FuncOf(replayInBrowser)
	js.Global().Set("echoevmReplay", replayFunction)
	js.Global().Set("echoevmWasmReady", true)
	select {}
}

func replayInBrowser(_ js.Value, args []js.Value) (encoded any) {
	defer func() {
		if recovered := recover(); recovered != nil {
			encoded = encodeResponse(browserResponse{Error: fmt.Sprintf("EchoEVM browser replay panicked: %v", recovered)})
		}
	}()
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return encodeResponse(browserResponse{Error: "replay witness JSON is required"})
	}
	options := browserOptions{Profile: "auto", Limit: replay.DefaultEvidenceLimit, MaxMemoryBytes: replay.DefaultEvidenceMemoryBytes}
	if len(args) > 1 && args[1].Type() == js.TypeString && strings.TrimSpace(args[1].String()) != "" {
		if err := json.Unmarshal([]byte(args[1].String()), &options); err != nil {
			return encodeResponse(browserResponse{Error: "decode browser replay options: " + err.Error()})
		}
	}
	if options.Profile == "" {
		options.Profile = "auto"
	}
	witness, err := replay.DecodeWitness(strings.NewReader(args[0].String()))
	if err != nil {
		return encodeResponse(browserResponse{Error: err.Error()})
	}
	result, err := replay.ReplayWitness(context.Background(), replay.ReplayRequest{
		Witness: witness, Profile: options.Profile, Limit: options.Limit, MaxMemoryBytes: options.MaxMemoryBytes,
	})
	if err != nil {
		return encodeResponse(browserResponse{Error: err.Error()})
	}
	browserResult := browserReplayResult{
		Transaction: result.Transaction,
		Execution: browserExecution{
			Engine: result.Execution.Engine, EngineVersion: result.Execution.EngineVersion,
			Status: string(result.Execution.Status), ReturnData: result.Execution.ReturnData,
			GasUsed: result.Execution.GasUsed, TotalSteps: len(result.Execution.Trace),
			StateEntries: len(result.State), Error: result.Execution.Error,
		},
		Warnings: result.Warnings, Witness: result.Witness, Evidence: result.Evidence,
	}
	return encodeResponse(browserResponse{OK: true, Result: &browserResult})
}

func encodeResponse(response browserResponse) string {
	data, err := json.Marshal(response)
	if err != nil {
		return `{"ok":false,"error":"encode EchoEVM browser response"}`
	}
	return string(data)
}
