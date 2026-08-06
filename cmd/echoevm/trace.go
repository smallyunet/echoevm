package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/config"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	explaintrace "github.com/smallyunet/echoevm/internal/trace"
	"github.com/spf13/cobra"
)

var traceFlags struct {
	binRuntime     string
	artifact       string
	calldata       string
	function       string
	args           string
	format         string
	profile        string
	fields         string
	opcodes        string
	limit          int
	depth          int
	fromStep       int
	toStep         int
	aroundStep     int
	window         int
	maxMemoryBytes int
	changesOnly    bool
	full           bool
}

func newTraceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trace",
		Short: "Execute runtime code and explain each opcode",
		Long:  "Execute runtime bytecode and emit AI-oriented opcode events with stack, memory, storage, gas, control-flow, and halt deltas.",
		RunE:  func(cmd *cobra.Command, args []string) error { return runTrace(cmd) },
		Example: strings.Join([]string{
			"echoevm trace -a ./Add.json -f 'add(uint256,uint256)' -A 1,2 --format text",
			"echoevm trace -r ./runtime.bin -d 0x1234 --opcodes SLOAD,SSTORE --changes-only",
			"echoevm trace -r ./runtime.bin -d 0x1234 --around-step 42 --window 5 --format json",
		}, "\n"),
	}
	cmd.Flags().StringVarP(&traceFlags.artifact, "artifact", "a", "", "Hardhat artifact JSON path")
	cmd.Flags().StringVarP(&traceFlags.binRuntime, "bin-runtime", "r", "", "Raw runtime bytecode (.bin)")
	cmd.Flags().StringVarP(&traceFlags.function, "function", "f", "", "Function signature")
	cmd.Flags().StringVarP(&traceFlags.args, "args", "A", "", "Comma separated function arguments")
	cmd.Flags().StringVarP(&traceFlags.calldata, "calldata", "d", "", "Full calldata hex")
	cmd.Flags().StringVar(&traceFlags.format, "format", "jsonl", "Trace format (jsonl|json|text|evidence-json)")
	cmd.Flags().StringVar(&traceFlags.profile, "profile", explaintrace.ProfileAuto, "Evidence profile for evidence-json (auto|revert|storage|call|abi|gas|full)")
	cmd.Flags().StringVar(&traceFlags.fields, "fields", "gas,stack,memory,storage,control,explanation", "Event fields to include")
	cmd.Flags().StringVar(&traceFlags.opcodes, "opcodes", "", "Comma-separated opcode names or hex bytes")
	cmd.Flags().IntVar(&traceFlags.limit, "limit", 0, "Maximum emitted opcode events; execution still completes (0 = no limit)")
	cmd.Flags().IntVar(&traceFlags.depth, "depth", -1, "Only emit one call depth (-1 = all depths)")
	cmd.Flags().IntVar(&traceFlags.fromStep, "from-step", 0, "First global opcode step to emit")
	cmd.Flags().IntVar(&traceFlags.toStep, "to-step", -1, "Last global opcode step to emit (-1 = no upper bound)")
	cmd.Flags().IntVar(&traceFlags.aroundStep, "around-step", -1, "Center output around one global opcode step")
	cmd.Flags().IntVar(&traceFlags.window, "window", 5, "Steps before and after --around-step")
	cmd.Flags().IntVar(&traceFlags.maxMemoryBytes, "max-memory-bytes", 256, "Maximum changed memory bytes captured per opcode")
	cmd.Flags().BoolVar(&traceFlags.changesOnly, "changes-only", false, "Only emit events with state, control-flow, or halt changes")
	cmd.Flags().BoolVar(&traceFlags.full, "full", false, "Deprecated: explainable events always pair pre/post state")
	_ = cmd.Flags().MarkDeprecated("full", "explainable trace events always include post-op deltas")
	return cmd
}

func runTrace(cmd *cobra.Command) error {
	if err := validateTraceFlags(); err != nil {
		return err
	}
	code, err := loadTraceCode()
	if err != nil {
		return err
	}
	calldata, err := traceCalldata()
	if err != nil {
		return err
	}

	gasLimit := uint64(config.DefaultGasLimit)
	intr := vm.NewWithCallData(code, calldata, core.NewMemoryStateDB(), common.Address{})
	intr.SetGas(gasLimit)
	intr.SetTraceDetails(true)
	collector := explaintrace.NewCollector(traceFlags.maxMemoryBytes)
	intr.RunWithHook(collector.Consume)
	allEvents := collector.Events()
	fields, err := parseTraceFields(traceFlags.fields)
	if err != nil {
		return err
	}
	filterLimit := traceFlags.limit
	if traceFlags.format == "evidence-json" {
		filterLimit = 0
	}
	filtered, matchedSteps := filterTraceEvents(allEvents, filterLimit)
	for index := range filtered {
		applyTraceFields(&filtered[index], fields)
	}

	status := "success"
	if intr.Err() != nil {
		status = "fault"
	} else if intr.IsReverted() {
		status = "revert"
	}
	execution := explaintrace.ExecutionResult{
		Status: status, GasLimit: gasLimit, GasUsed: gasLimit - intr.Gas(),
		ReturnData: fmt.Sprintf("0x%x", intr.ReturnedCode()), TotalSteps: len(allEvents),
		MatchedSteps: matchedSteps, EmittedSteps: len(filtered), Filtered: matchedSteps < len(allEvents),
		Truncated: len(filtered) < matchedSteps,
	}
	if intr.Err() != nil {
		execution.ExecutionError = intr.Err().Error()
	}
	if traceFlags.format == "evidence-json" {
		document, evidenceErr := explaintrace.BuildEvidence(execution, filtered, traceFlags.profile, traceFlags.limit)
		if evidenceErr != nil {
			return evidenceErr
		}
		if err := writeEvidence(cmd, document); err != nil {
			return err
		}
	} else {
		document := explaintrace.Document{Schema: explaintrace.SchemaVersion, Execution: execution, Events: filtered}
		if err := writeTrace(cmd, document); err != nil {
			return err
		}
	}
	if intr.Err() != nil {
		return fmt.Errorf("execution failed: %w", intr.Err())
	}
	return nil
}

func validateTraceFlags() error {
	switch traceFlags.format {
	case "jsonl", "json", "text", "evidence-json":
	default:
		return fmt.Errorf("unsupported trace format %q (want jsonl, json, text, or evidence-json)", traceFlags.format)
	}
	if traceFlags.format == "evidence-json" {
		if err := explaintrace.ValidateEvidenceProfile(traceFlags.profile); err != nil {
			return err
		}
	}
	if traceFlags.depth < -1 || traceFlags.fromStep < 0 || traceFlags.toStep < -1 || traceFlags.aroundStep < -1 || traceFlags.window < 0 || traceFlags.limit < 0 || traceFlags.maxMemoryBytes < 0 {
		return fmt.Errorf("trace numeric filters must be non-negative, except -1 sentinel values")
	}
	if traceFlags.toStep >= 0 && traceFlags.toStep < traceFlags.fromStep {
		return fmt.Errorf("--to-step must be greater than or equal to --from-step")
	}
	return nil
}

func loadTraceCode() ([]byte, error) {
	if traceFlags.artifact == "" && traceFlags.binRuntime == "" {
		return nil, fmt.Errorf("one of --artifact or --bin-runtime must be provided")
	}
	if traceFlags.artifact != "" && traceFlags.binRuntime != "" {
		return nil, fmt.Errorf("provide only one of --artifact or --bin-runtime")
	}
	var runtimeHex string
	if traceFlags.artifact != "" {
		data, err := os.ReadFile(traceFlags.artifact)
		if err != nil {
			return nil, err
		}
		var artifact struct {
			DeployedBytecode string `json:"deployedBytecode"`
			Bytecode         string `json:"bytecode"`
		}
		if err := json.Unmarshal(data, &artifact); err != nil {
			return nil, err
		}
		runtimeHex = artifact.DeployedBytecode
		if runtimeHex == "" || runtimeHex == "0x" {
			runtimeHex = artifact.Bytecode
		}
	} else {
		data, err := os.ReadFile(traceFlags.binRuntime)
		if err != nil {
			return nil, err
		}
		runtimeHex = string(data)
	}
	runtimeHex = strings.TrimPrefix(strings.TrimSpace(runtimeHex), "0x")
	code, err := hex.DecodeString(runtimeHex)
	if err != nil {
		return nil, fmt.Errorf("invalid runtime bytecode: %w", err)
	}
	return code, nil
}

func traceCalldata() ([]byte, error) {
	if traceFlags.calldata != "" && traceFlags.function != "" {
		return nil, fmt.Errorf("provide only one of --calldata or --function")
	}
	if traceFlags.calldata != "" {
		value, err := hex.DecodeString(strings.TrimPrefix(traceFlags.calldata, "0x"))
		if err != nil {
			return nil, fmt.Errorf("invalid calldata: %w", err)
		}
		return value, nil
	}
	if traceFlags.function != "" {
		return buildCallData(traceFlags.function, traceFlags.args)
	}
	return nil, fmt.Errorf("provide --calldata or --function + --args")
}

func filterTraceEvents(events []explaintrace.OpcodeEvent, limit int) ([]explaintrace.OpcodeEvent, int) {
	opcodes := make(map[string]struct{})
	for _, value := range strings.Split(traceFlags.opcodes, ",") {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value != "" {
			opcodes[value] = struct{}{}
		}
	}
	from, to := traceFlags.fromStep, traceFlags.toStep
	if traceFlags.aroundStep >= 0 {
		from = max(from, traceFlags.aroundStep-traceFlags.window)
		aroundEnd := traceFlags.aroundStep + traceFlags.window
		if to < 0 || aroundEnd < to {
			to = aroundEnd
		}
	}
	filtered := make([]explaintrace.OpcodeEvent, 0, len(events))
	matched := 0
	for _, event := range events {
		if event.Step < from || (to >= 0 && event.Step > to) {
			continue
		}
		if traceFlags.depth >= 0 && event.Depth != traceFlags.depth {
			continue
		}
		if len(opcodes) > 0 {
			if _, ok := opcodes[strings.ToUpper(event.OpcodeName)]; !ok {
				if _, hexOK := opcodes[strings.ToUpper(event.Opcode)]; !hexOK {
					continue
				}
			}
		}
		if traceFlags.changesOnly && !traceEventChanged(event) {
			continue
		}
		matched++
		if limit == 0 || len(filtered) < limit {
			filtered = append(filtered, event)
		}
	}
	return filtered, matched
}

func writeEvidence(cmd *cobra.Command, document explaintrace.EvidenceDocument) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetEscapeHTML(false)
	return encoder.Encode(document)
}

func traceEventChanged(event explaintrace.OpcodeEvent) bool {
	return event.Halt || event.Error != "" || event.Control != nil || event.Memory != nil || len(event.Storage) > 0 ||
		(event.Stack != nil && (event.Stack.Reordered || len(event.Stack.Popped) > 0 || len(event.Stack.Pushed) > 0))
}

type traceFieldSet map[string]bool

func parseTraceFields(raw string) (traceFieldSet, error) {
	allowed := map[string]bool{"gas": true, "stack": true, "memory": true, "storage": true, "control": true, "explanation": true}
	fields := make(traceFieldSet)
	for _, value := range strings.Split(raw, ",") {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if !allowed[value] {
			keys := make([]string, 0, len(allowed))
			for key := range allowed {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			return nil, fmt.Errorf("unsupported trace field %q (want %s)", value, strings.Join(keys, ","))
		}
		fields[value] = true
	}
	return fields, nil
}

func applyTraceFields(event *explaintrace.OpcodeEvent, fields traceFieldSet) {
	if !fields["gas"] {
		event.Gas = nil
	}
	if !fields["stack"] {
		event.Stack = nil
	}
	if !fields["memory"] {
		event.Memory = nil
	}
	if !fields["storage"] {
		event.Storage = nil
	}
	if !fields["control"] {
		event.Control = nil
	}
	if !fields["explanation"] {
		event.Explanation = ""
	}
}

func writeTrace(cmd *cobra.Command, document explaintrace.Document) error {
	out := cmd.OutOrStdout()
	switch traceFlags.format {
	case "json":
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(document)
	case "jsonl":
		encoder := json.NewEncoder(out)
		for _, event := range document.Events {
			if err := encoder.Encode(event); err != nil {
				return err
			}
		}
		return encoder.Encode(struct {
			Type      string                       `json:"type"`
			Schema    string                       `json:"schema"`
			Execution explaintrace.ExecutionResult `json:"execution"`
		}{Type: "result", Schema: explaintrace.SchemaVersion, Execution: document.Execution})
	case "text":
		for _, event := range document.Events {
			if _, err := fmt.Fprintf(out, "#%d d%d pc=%d %-12s", event.Step, event.Depth, event.PC, event.OpcodeName); err != nil {
				return err
			}
			if event.Gas != nil {
				if _, err := fmt.Fprintf(out, " gas=%d->%d (-%d)", event.Gas.Before, event.Gas.After, event.Gas.Used); err != nil {
					return err
				}
			}
			if event.Stack != nil && (event.Stack.Reordered || len(event.Stack.Popped) > 0 || len(event.Stack.Pushed) > 0) {
				stackText := fmt.Sprintf("-%v +%v", event.Stack.Popped, event.Stack.Pushed)
				if event.Stack.Reordered {
					stackText = fmt.Sprintf("%v->%v", event.Stack.TopBefore, event.Stack.TopAfter)
				}
				if _, err := fmt.Fprintf(out, " stack %s", stackText); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			if event.Explanation != "" {
				if _, err := fmt.Fprintf(out, "  %s\n", event.Explanation); err != nil {
					return err
				}
			}
		}
		_, err := fmt.Fprintf(out, "result status=%s gas=%d/%d steps=%d matched=%d emitted=%d truncated=%t return=%s\n",
			document.Execution.Status, document.Execution.GasUsed, document.Execution.GasLimit,
			document.Execution.TotalSteps, document.Execution.MatchedSteps, document.Execution.EmittedSteps,
			document.Execution.Truncated, document.Execution.ReturnData)
		return err
	default:
		return fmt.Errorf("unsupported trace format %q", traceFlags.format)
	}
}
