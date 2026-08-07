package trace

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBuildEvidenceAutoDropsStackPlumbingAndGas(t *testing.T) {
	events := []OpcodeEvent{
		{Step: 0, PC: 0, Opcode: "0x60", OpcodeName: "PUSH1", Address: common.Address{}.Hex(), Gas: &GasDelta{Before: 100, After: 97}},
		{Step: 1, PC: 2, Opcode: "0x04", OpcodeName: "DIV", Address: common.Address{}.Hex(), Gas: &GasDelta{Before: 97, After: 92}, Stack: &StackDelta{Popped: []string{"0x2", "0x8"}, Pushed: []string{"0x0"}}, Explanation: "divide"},
		{Step: 2, PC: 3, Opcode: "0xf3", OpcodeName: "RETURN", Address: common.Address{}.Hex(), Control: &ControlFlow{Kind: "return"}, Halt: true},
	}
	document, err := BuildEvidence(ExecutionResult{Status: "success", GasUsed: 8, TotalSteps: 3}, events, ProfileAuto, 40)
	if err != nil {
		t.Fatal(err)
	}
	if document.Schema != EvidenceSchemaVersion || len(document.Events) != 2 || document.Events[0].Op != "DIV" {
		t.Fatalf("document = %+v", document)
	}
	if document.Events[0].Gas != nil || document.Events[0].Address != "" || document.Events[0].Stack == nil {
		t.Fatalf("compact event = %+v", document.Events[0])
	}
	if document.Selection.Omitted != 1 || document.Selection.Truncated {
		t.Fatalf("selection = %+v", document.Selection)
	}
}

func TestBuildEvidenceLimitRetainsFault(t *testing.T) {
	events := []OpcodeEvent{
		{Step: 0, PC: 0, Opcode: "0x01", OpcodeName: "ADD"},
		{Step: 1, PC: 1, Opcode: "0x02", OpcodeName: "MUL"},
		{Step: 2, PC: 2, Opcode: "0x56", OpcodeName: "JUMP", Halt: true, Error: "invalid jump destination"},
	}
	document, err := BuildEvidence(ExecutionResult{Status: "fault", TotalSteps: 3}, events, ProfileAuto, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Events) != 1 || document.Events[0].Op != "JUMP" || !document.Selection.Truncated || document.Selection.Omitted != 2 {
		t.Fatalf("document = %+v", document)
	}
}

func TestBuildEvidenceStorageProfileKeepsStateAndTerminalControl(t *testing.T) {
	events := []OpcodeEvent{
		{Step: 0, Opcode: "0x04", OpcodeName: "DIV"},
		{Step: 1, Opcode: "0x55", OpcodeName: "SSTORE", Storage: []StorageAccess{{Kind: "write"}}},
		{Step: 2, Opcode: "0xf3", OpcodeName: "RETURN", Control: &ControlFlow{Kind: "return"}, Halt: true},
	}
	document, err := BuildEvidence(ExecutionResult{Status: "success", TotalSteps: 3}, events, ProfileStorage, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Events) != 2 || document.Events[0].Op != "SSTORE" || document.Events[1].Op != "RETURN" {
		t.Fatalf("events = %+v", document.Events)
	}
}

func TestBuildEvidenceCompactsStorageWordsAndDefaults(t *testing.T) {
	applied := true
	events := []OpcodeEvent{{
		Step: 0, Opcode: "0x55", OpcodeName: "SSTORE", Address: common.Address{}.Hex(),
		Storage: []StorageAccess{{
			Kind: "write", Address: common.Address{}.Hex(),
			Slot:           "0x0000000000000000000000000000000000000000000000000000000000000000",
			Before:         "0x0000000000000000000000000000000000000000000000000000000000000000",
			After:          "0x000000000000000000000000000000000000000000000000000000000000002a",
			Original:       "0x0000000000000000000000000000000000000000000000000000000000000000",
			AppliedInFrame: &applied,
		}},
	}}
	document, err := BuildEvidence(ExecutionResult{Status: "success", TotalSteps: 1}, events, ProfileAuto, 0)
	if err != nil {
		t.Fatal(err)
	}
	access := document.Events[0].Storage[0]
	if access.Address != "" || access.Slot != "0x0" || access.Before != "0x0" || access.After != "0x2a" || access.Original != "" {
		t.Fatalf("access = %+v", access)
	}
}

func TestValidateEvidenceProfileRejectsUnknownValue(t *testing.T) {
	if err := ValidateEvidenceProfile("mystery"); err == nil {
		t.Fatal("expected unknown profile to fail")
	}
}

func TestBuildEvidenceLinksNestedFrameAndRollback(t *testing.T) {
	events := []OpcodeEvent{
		{Step: 5, Depth: 0, PC: 10, Opcode: "0xf1", OpcodeName: "CALL", Control: &ControlFlow{Kind: "call"}},
		{Step: 6, Depth: 1, PC: 0, Opcode: "0x55", OpcodeName: "SSTORE", Storage: []StorageAccess{{Kind: "write"}}},
		{Step: 7, Depth: 1, PC: 1, Opcode: "0xfd", OpcodeName: "REVERT", Control: &ControlFlow{Kind: "revert"}, Reverted: true, Halt: true},
		{Step: 8, Depth: 0, PC: 11, Opcode: "0x15", OpcodeName: "ISZERO"},
	}
	document, err := BuildEvidence(ExecutionResult{Status: "success", TotalSteps: 9}, events, ProfileAuto, 40)
	if err != nil {
		t.Fatal(err)
	}
	wantKinds := []string{"enters-frame", "returns-to", "rolls-back"}
	if len(document.Links) != len(wantKinds) {
		t.Fatalf("links = %+v", document.Links)
	}
	for index, kind := range wantKinds {
		if document.Links[index].Kind != kind {
			t.Fatalf("link %d = %+v, want %s", index, document.Links[index], kind)
		}
	}
	if document.Links[2].From.Op != "SSTORE" || document.Links[2].To.Op != "REVERT" {
		t.Fatalf("rollback link = %+v", document.Links[2])
	}
}

func TestBuildEvidenceLinksValueFlowThroughDupAndSwap(t *testing.T) {
	events := []OpcodeEvent{
		{Step: 0, PC: 0, Opcode: "0x60", OpcodeName: "PUSH1", Stack: &StackDelta{SizeBefore: 0, SizeAfter: 1, Pushed: []string{"0x8"}}},
		{Step: 1, PC: 2, Opcode: "0x60", OpcodeName: "PUSH1", Stack: &StackDelta{SizeBefore: 1, SizeAfter: 2, Pushed: []string{"0x2"}}},
		{Step: 2, PC: 4, Opcode: "0x03", OpcodeName: "SUB", Stack: &StackDelta{SizeBefore: 2, SizeAfter: 1, Popped: []string{"0x2", "0x8"}, Pushed: []string{"0x6"}}},
		{Step: 3, PC: 5, Opcode: "0x80", OpcodeName: "DUP1", Stack: &StackDelta{SizeBefore: 1, SizeAfter: 2, Pushed: []string{"0x6"}}},
		{Step: 4, PC: 6, Opcode: "0x90", OpcodeName: "SWAP1", Stack: &StackDelta{SizeBefore: 2, SizeAfter: 2, Reordered: true}},
		{Step: 5, PC: 7, Opcode: "0x04", OpcodeName: "DIV", Stack: &StackDelta{SizeBefore: 2, SizeAfter: 1, Popped: []string{"0x6", "0x6"}, Pushed: []string{"0x1"}}},
	}
	document, err := BuildEvidence(ExecutionResult{Status: "success", TotalSteps: 6}, events, ProfileMath, 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Events) != 2 || document.Events[0].Op != "SUB" || document.Events[1].Op != "DIV" {
		t.Fatalf("events = %+v", document.Events)
	}
	if len(document.Links) != 2 {
		t.Fatalf("links = %+v", document.Links)
	}
	for _, link := range document.Links {
		if link.Kind != "value-flow" || link.From.Op != "SUB" || link.To.Op != "DIV" || link.Value != "0x6" {
			t.Fatalf("link = %+v", link)
		}
	}
}
