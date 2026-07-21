package differential

import (
	"context"
	"testing"
)

func TestDefaultEngineInitialStorageAndTrace(t *testing.T) {
	result, err := DefaultEngine().Compare(context.Background(), Request{
		Fork: ForkCancun, Bytecode: "5f545f5260205ff3", Calldata: "0x", GasLimit: DefaultGasLimit,
		InitialStorage: map[string]string{"0x0": "0x2a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Match {
		t.Fatalf("unexpected divergence: %+v", result.FirstDivergence)
	}
	want := "0x000000000000000000000000000000000000000000000000000000000000002a"
	if result.EchoEVM.ReturnData != want {
		t.Fatalf("return=%s want %s", result.EchoEVM.ReturnData, want)
	}
	if len(result.EchoEVM.Trace) == 0 {
		t.Fatal("missing normalized trace")
	}
	last := result.EchoEVM.Trace[len(result.EchoEVM.Trace)-1]
	if last.StackAfter != nil {
		t.Fatalf("terminal stack must be omitted, got %v", last.StackAfter)
	}
}

func TestExecutionStatusesAreComparableResults(t *testing.T) {
	for _, test := range []struct {
		name, code string
		status     Status
	}{
		{"success", "00", StatusSuccess}, {"revert", "5f5ffd", StatusRevert}, {"fault", "fe", StatusFault},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := DefaultEngine().Compare(context.Background(), Request{Fork: ForkCancun, Bytecode: test.code, Calldata: "0x", GasLimit: DefaultGasLimit})
			if err != nil {
				t.Fatal(err)
			}
			if !result.Match || result.EchoEVM.Status != test.status {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}
