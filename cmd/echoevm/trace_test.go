package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	explaintrace "github.com/smallyunet/echoevm/internal/trace"
)

func TestTraceLimitDoesNotStopExecution(t *testing.T) {
	runtimePath := filepath.Join(t.TempDir(), "runtime.bin")
	if err := os.WriteFile(runtimePath, []byte("600160020100"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newTraceCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--bin-runtime", runtimePath, "--calldata", "0x", "--limit", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(&output)
	var records []map[string]any
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("invalid JSONL %q: %v", scanner.Text(), err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[0]["type"] != "opcode" || records[1]["type"] != "result" {
		t.Fatalf("records = %#v", records)
	}
	execution := records[1]["execution"].(map[string]any)
	if execution["totalSteps"] != float64(4) || execution["matchedSteps"] != float64(4) || execution["emittedSteps"] != float64(1) || execution["truncated"] != true {
		t.Fatalf("execution = %#v", execution)
	}
	if execution["gasUsed"] != float64(9) {
		t.Fatalf("gasUsed = %#v, want 9", execution["gasUsed"])
	}
}

func TestTraceJSONSupportsOpcodeAndFieldFilters(t *testing.T) {
	runtimePath := filepath.Join(t.TempDir(), "runtime.bin")
	if err := os.WriteFile(runtimePath, []byte("600160020100"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newTraceCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"--bin-runtime", runtimePath, "--calldata", "0x", "--format", "json",
		"--opcodes", "ADD", "--fields", "stack,explanation",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var document explaintrace.Document
	if err := json.Unmarshal(output.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.Schema != explaintrace.SchemaVersion || len(document.Events) != 1 || document.Events[0].OpcodeName != "ADD" {
		t.Fatalf("document = %+v", document)
	}
	event := document.Events[0]
	if event.Gas != nil || event.Stack == nil || event.Explanation == "" {
		t.Fatalf("filtered event = %+v", event)
	}
}
