package differential

import (
	"context"
	"encoding/hex"
	"testing"
)

func TestDefaultEngineInitialStorageAndTrace(t *testing.T) {
	result, err := DefaultEngine().RunEcho(context.Background(), Request{
		Fork: ForkCancun, Bytecode: "5f545f5260205ff3", Calldata: "0x", GasLimit: DefaultGasLimit,
		InitialStorage: map[string]string{"0x0": "0x2a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "0x000000000000000000000000000000000000000000000000000000000000002a"
	if result.ReturnData != want {
		t.Fatalf("return=%s want %s", result.ReturnData, want)
	}
	if len(result.Trace) == 0 {
		t.Fatal("missing normalized trace")
	}
	last := result.Trace[len(result.Trace)-1]
	if last.StackAfter != nil {
		t.Fatalf("terminal stack must be omitted, got %v", last.StackAfter)
	}
}

func TestExecutionStatuses(t *testing.T) {
	for _, test := range []struct {
		name, code string
		status     Status
	}{
		{"success", "00", StatusSuccess}, {"revert", "5f5ffd", StatusRevert}, {"fault", "fe", StatusFault},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := DefaultEngine().RunEcho(context.Background(), Request{Fork: ForkCancun, Bytecode: test.code, Calldata: "0x", GasLimit: DefaultGasLimit})
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != test.status {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

func TestEngineDeploysInitcodeBeforeCall(t *testing.T) {
	// Constructor stores 7 in slot zero, deploys a runtime that returns slot zero.
	const runtime = "5f545f5260205ff3"
	const initcode = "60075f556008600e5f3960085ff3" + runtime
	result, err := DefaultEngine().RunEcho(context.Background(), Request{
		Fork: ForkCancun, Bytecode: runtime, InitCode: initcode,
		Calldata: "0x", GasLimit: 100_000,
	})
	if err != nil {
		t.Fatalf("run deployed contract: %v", err)
	}
	const seven = "0x0000000000000000000000000000000000000000000000000000000000000007"
	if result.ReturnData != seven {
		t.Fatalf("return data = %s, want %s", result.ReturnData, seven)
	}
}

func TestEngineUsesSeparateDeploymentAndCallGasLimits(t *testing.T) {
	// The constructor's SSTORE needs more gas than the deliberately tight
	// runtime call. DeployGasLimit must not weaken the call-gas assertion.
	const runtime = "5f545f5260205ff3"
	const initcode = "60075f556008600e5f3960085ff3" + runtime
	result, err := DefaultEngine().RunEcho(context.Background(), Request{
		Fork: ForkCancun, Bytecode: runtime, InitCode: initcode,
		Calldata: "0x", GasLimit: 10_000, DeployGasLimit: 100_000,
	})
	if err != nil {
		t.Fatalf("run with separate gas limits: %v", err)
	}
	if result.Status != StatusSuccess {
		t.Fatalf("unexpected separate-gas result: %+v", result)
	}
}

func TestRunEchoExplainIncludesNestedStorageAndRevert(t *testing.T) {
	childCode := []byte{0x60, 0x01, 0x5f, 0x55, 0x5f, 0x5f, 0xfd}
	parentLabel := byte(5 + len(childCode))
	code := []byte{0x36, 0x15, 0x60, parentLabel, 0x57}
	code = append(code, childCode...)
	code = append(code,
		0x5b, 0x60, 0x01, 0x5f, 0x53,
		0x5f, 0x5f, 0x60, 0x01, 0x5f, 0x5f, 0x30, 0x61, 0xff, 0xff, 0xf1,
		0x5f, 0x52, 0x60, 0x20, 0x5f, 0xf3,
	)
	result, events, err := DefaultEngine().RunEchoExplain(context.Background(), Request{
		Fork: ForkCancun, Bytecode: hex.EncodeToString(code), Calldata: "0x", GasLimit: DefaultGasLimit,
	}, 256)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusSuccess {
		t.Fatalf("result = %+v", result)
	}
	foundWrite, foundRevert := false, false
	for _, event := range events {
		if event.Depth == 1 && event.OpcodeName == "SSTORE" && len(event.Storage) == 1 {
			foundWrite = true
		}
		if event.Depth == 1 && event.OpcodeName == "REVERT" && event.Reverted {
			foundRevert = true
		}
	}
	if !foundWrite || !foundRevert {
		t.Fatalf("nested evidence missing write=%t revert=%t events=%+v", foundWrite, foundRevert, events)
	}
}
