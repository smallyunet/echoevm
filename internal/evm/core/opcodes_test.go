package core

import "testing"

func TestOpcodeName(t *testing.T) {
	tests := []struct {
		op   byte
		name string
	}{
		{ADD, "ADD"},
		{PUSH1, "PUSH1"},
		{DUP1, "DUP1"},
		{SWAP1, "SWAP1"},
		{0xff, "SELFDESTRUCT"},
		{0xfe, "INVALID"},
		{0xaa, "UNKNOWN"},
	}
	for _, tt := range tests {
		if got := OpcodeName(tt.op); got != tt.name {
			t.Fatalf("op 0x%02x want %s got %s", tt.op, tt.name, got)
		}
	}
	if OpcodeName(0xef) != "UNKNOWN" {
		t.Fatalf("unexpected name for unknown opcode")
	}
}
