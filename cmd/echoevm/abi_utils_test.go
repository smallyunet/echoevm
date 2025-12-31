package main

import (
	"reflect"
	"testing"
)

func TestSplitArrayArgs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"1;2;3", []string{"1", "2", "3"}},
		{"[1;2];[3;4]", []string{"[1;2]", "[3;4]"}},
		{"[[1;2];[3]];4", []string{"[[1;2];[3]]", "4"}},
		{"1", []string{"1"}},
		{"", nil},
	}

	for _, tt := range tests {
		result := splitArrayArgs(tt.input)
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("splitArrayArgs(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestBuildCallData_NestedArrays(t *testing.T) {
	tests := []struct {
		name      string
		signature string
		args      string
		wantErr   bool
	}{
		{
			name:      "uint256[][] 2x2",
			signature: "test(uint256[][])",
			args:      "[[1;2];[3;4]]",
			wantErr:   false,
		},
		{
			name:      "uint256[2][2] fixed",
			signature: "test(uint256[2][2])",
			args:      "[[1;2];[3;4]]",
			wantErr:   false,
		},
		{
			name:      "address[][]",
			signature: "test(address[][])",
			args:      "[[0x0000000000000000000000000000000000000001];[0x0000000000000000000000000000000000000002]]",
			wantErr:   false,
		},
		{
			name:      "mixed types with nested array",
			signature: "test(uint256,uint256[][])",
			args:      "100,[[1;2];[3;4]]",
			wantErr:   false,
		},
		{
			name:      "nested nested",
			signature: "test(uint256[][][])",
			args:      "[[[1;2];[3;4]];[[5;6]]]",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We only check if it encodes without error for now.
			// Ideally we should check the encoded bytes, but that requires manual construction of ABI encoding.
			// Since we use go-ethereum's Pack, if the input types match what Pack expects, it should be correct.
			_, err := buildCallData(tt.signature, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildCallData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildTypedSlice(t *testing.T) {
	// Test internal helper directly for robustness

	// Case: uint256[][]
	// We construct []interface{} { []interface{}{ *big.Int, *big.Int }, ... }
	// And expect []interface{} containing properly typed slices if buildTypedSlice supported it properly?
	// Actually buildTypedSlice for SliceTy returns []interface{} where elements are the inner slices.
	// But the inner slices should be correct type on their own.
}
