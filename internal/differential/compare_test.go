package differential

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type fakeRunner struct {
	result ExecutionResult
	err    error
}

func (f fakeRunner) Run(context.Context, Request) (ExecutionResult, error) { return f.result, f.err }

func baseResult(engine string) ExecutionResult {
	return ExecutionResult{Engine: engine, Status: StatusSuccess, ReturnData: "0x", GasUsed: 3,
		Storage: map[string]string{}, Trace: []NormalizedStep{{Index: 0, PC: 0, Opcode: "0x60", OpcodeName: "PUSH1", GasBefore: 100, GasAfter: 97, StackBefore: []string{}, StackAfter: []string{"0x1"}}}}
}

func compareMutated(t *testing.T, mutate func(*ExecutionResult)) ComparisonResult {
	t.Helper()
	echo, geth := baseResult("EchoEVM"), baseResult("Geth")
	mutate(&geth)
	return CompareResults(Request{Fork: ForkCancun}, echo, geth)
}

func requireDivergence(t *testing.T, result ComparisonResult, field string, step *int) {
	t.Helper()
	if result.Match || result.FirstDivergence == nil {
		t.Fatal("expected divergence")
	}
	if result.FirstDivergence.Field != field {
		t.Fatalf("field=%s want %s", result.FirstDivergence.Field, field)
	}
	if step == nil && result.FirstDivergence.Step != nil {
		t.Fatalf("unexpected step %d", *result.FirstDivergence.Step)
	}
	if step != nil && (result.FirstDivergence.Step == nil || *result.FirstDivergence.Step != *step) {
		t.Fatalf("step=%v want %d", result.FirstDivergence.Step, *step)
	}
}

func TestResultDivergences(t *testing.T) {
	tests := []struct {
		name, field string
		mutate      func(*ExecutionResult)
	}{
		{"gas", "gasUsed", func(r *ExecutionResult) { r.GasUsed++ }},
		{"return", "returnData", func(r *ExecutionResult) { r.ReturnData = "0x01" }},
		{"success-revert", "status", func(r *ExecutionResult) { r.Status = StatusRevert }},
		{"success-fault", "status", func(r *ExecutionResult) { r.Status = StatusFault }},
		{"revert-fault", "status", func(r *ExecutionResult) { r.Status = StatusFault }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := compareMutated(t, test.mutate)
			requireDivergence(t, result, test.field, nil)
		})
	}
}

func TestRevertFaultDivergence(t *testing.T) {
	echo, geth := baseResult("EchoEVM"), baseResult("Geth")
	echo.Status, geth.Status = StatusRevert, StatusFault
	result := CompareResults(Request{Fork: ForkCancun}, echo, geth)
	requireDivergence(t, result, "status", nil)
}

func TestTraceDivergencesLocateFirstStep(t *testing.T) {
	step := 0
	tests := []struct {
		name, field string
		mutate      func(*ExecutionResult)
	}{
		{"stack", "stackBefore", func(r *ExecutionResult) { r.Trace[0].StackBefore = []string{"0x2"} }},
		{"pc", "pc", func(r *ExecutionResult) { r.Trace[0].PC = 1 }},
		{"opcode", "opcode", func(r *ExecutionResult) { r.Trace[0].Opcode = "0x61" }},
		{"length", "length", func(r *ExecutionResult) { r.Trace = append(r.Trace, r.Trace[0]); r.Trace[1].Index = 1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := compareMutated(t, test.mutate)
			wantStep := step
			if test.field == "length" {
				wantStep = 1
			}
			requireDivergence(t, result, test.field, &wantStep)
		})
	}
}

func TestTraceDivergencePrecedesFinalResultDifference(t *testing.T) {
	result := compareMutated(t, func(r *ExecutionResult) { r.GasUsed++; r.Trace[0].GasAfter++ })
	step := 0
	requireDivergence(t, result, "gasAfter", &step)
}

func TestEngineValidationAndRunnerErrors(t *testing.T) {
	valid := Request{Fork: ForkCancun, Bytecode: "00", Calldata: "0x", GasLimit: 1}
	engine := NewEngine(fakeRunner{result: baseResult("EchoEVM")}, fakeRunner{result: baseResult("Geth")})
	for _, req := range []Request{{Bytecode: ""}, {Bytecode: "0x0"}, {Bytecode: "zz"}, {Bytecode: "00", GasLimit: MaxGasLimit + 1}} {
		if _, err := engine.Compare(context.Background(), req); err == nil {
			t.Fatalf("expected validation error for %+v", req)
		}
	}
	broken := NewEngine(fakeRunner{err: errors.New("boom")}, fakeRunner{})
	if _, err := broken.Compare(context.Background(), valid); err == nil {
		t.Fatal("expected runner error")
	}
}

func TestComparisonJSONIsStableAndParseable(t *testing.T) {
	result := CompareResults(Request{Fork: ForkCancun, Bytecode: "0x00", Calldata: "0x", GasLimit: 1}, baseResult("EchoEVM"), baseResult("Geth"))
	a, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatal("JSON encoding changed between identical marshals")
	}
	var decoded ComparisonResult
	if err := json.Unmarshal(a, &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.Match {
		t.Fatal("decoded match=false")
	}
}
