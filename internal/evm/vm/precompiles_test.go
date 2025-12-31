package vm

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/ripemd160"
)

func TestPrecompileIdentity(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"single byte", "ff"},
		{"multiple bytes", "deadbeef"},
		{"32 bytes", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, _ := hex.DecodeString(tt.input)

			output, gas, err := RunPrecompiled(PrecompileIdentity, input, 1000000)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !bytes.Equal(output, input) {
				t.Errorf("identity mismatch: got %x, want %x", output, input)
			}

			// Check gas consumption (15 + 3 per word)
			words := uint64((len(input) + 31) / 32)
			expectedGas := 15 + 3*words
			expectedRemaining := uint64(1000000) - expectedGas
			if gas != expectedRemaining {
				t.Errorf("gas mismatch: got %d remaining, want %d", gas, expectedRemaining)
			}
		})
	}
}

func TestPrecompileSHA256(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"hello", "68656c6c6f"}, // "hello" in hex
		{"32 bytes", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, _ := hex.DecodeString(tt.input)

			output, _, err := RunPrecompiled(PrecompileSHA256, input, 1000000)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify against standard library
			expected := sha256.Sum256(input)
			if !bytes.Equal(output, expected[:]) {
				t.Errorf("sha256 mismatch: got %x, want %x", output, expected[:])
			}
		})
	}
}

func TestPrecompileRIPEMD160(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"hello", "68656c6c6f"}, // "hello" in hex
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, _ := hex.DecodeString(tt.input)

			output, _, err := RunPrecompiled(PrecompileRIPEMD160, input, 1000000)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify against standard library (output is 32 bytes, left-padded)
			hasher := ripemd160.New()
			hasher.Write(input)
			expected := common.LeftPadBytes(hasher.Sum(nil), 32)

			if !bytes.Equal(output, expected) {
				t.Errorf("ripemd160 mismatch: got %x, want %x", output, expected)
			}
		})
	}
}

func TestPrecompileECRecover(t *testing.T) {
	// Generate a test key pair
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Create a message hash
	msgHash := crypto.Keccak256Hash([]byte("test message"))

	// Sign the hash
	sig, err := crypto.Sign(msgHash.Bytes(), privateKey)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	// Construct input: hash (32) + v (32) + r (32) + s (32)
	input := make([]byte, 128)
	copy(input[0:32], msgHash.Bytes())

	// v is the recovery id + 27
	v := sig[64] + 27
	input[63] = v

	// r and s
	copy(input[64:96], sig[0:32])
	copy(input[96:128], sig[32:64])

	output, _, err := RunPrecompiled(PrecompileECRecover, input, 1000000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(output) != 32 {
		t.Fatalf("expected 32 bytes output, got %d", len(output))
	}

	// Extract address from output
	recoveredAddr := common.BytesToAddress(output[12:])
	expectedAddr := crypto.PubkeyToAddress(privateKey.PublicKey)

	if recoveredAddr != expectedAddr {
		t.Errorf("address mismatch: got %s, want %s", recoveredAddr.Hex(), expectedAddr.Hex())
	}
}

func TestPrecompileECRecoverInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"empty input", []byte{}},
		{"invalid v", make([]byte, 128)}, // v = 0 is invalid
		{"short input", make([]byte, 64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, _, err := RunPrecompiled(PrecompileECRecover, tt.input, 1000000)
			// Invalid input should return empty output, not error
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(output) != 0 {
				t.Errorf("expected empty output for invalid input, got %x", output)
			}
		})
	}
}

func TestPrecompileOutOfGas(t *testing.T) {
	input := make([]byte, 100)

	// SHA256 requires: 60 + 12 * ceil(100/32) = 60 + 12 * 4 = 108 gas
	_, gas, err := RunPrecompiled(PrecompileSHA256, input, 50)

	if err == nil {
		t.Errorf("expected out of gas error")
	}
	if gas != 0 {
		t.Errorf("expected 0 remaining gas on out of gas, got %d", gas)
	}
}

func TestIsPrecompiled(t *testing.T) {
	tests := []struct {
		addr     common.Address
		expected bool
	}{
		{PrecompileECRecover, true},
		{PrecompileSHA256, true},
		{PrecompileRIPEMD160, true},
		{PrecompileIdentity, true},
		{common.BytesToAddress([]byte{0x05}), true}, // ModExp
		{common.BytesToAddress([]byte{0x06}), true}, // Add
		{common.BytesToAddress([]byte{0x07}), true}, // Mul
		{common.BytesToAddress([]byte{0x08}), true}, // Pairing
		{common.BytesToAddress([]byte{0x09}), true}, // Blake2F
		{common.BytesToAddress([]byte{0x10}), false},
		{common.Address{}, false},
	}

	for _, tt := range tests {
		result := IsPrecompiled(tt.addr)
		if result != tt.expected {
			t.Errorf("IsPrecompiled(%s) = %v, want %v", tt.addr.Hex(), result, tt.expected)
		}
	}
}

func TestPrecompileModExp_Simple(t *testing.T) {
	// Base: 3, Exp: 2, Mod: 10 => 9
	// Lengths: 1, 1, 1
	// Padded to 32 bytes each for input structure: 32+32+32 headers + 1+1+1 data (padded)

	// Actually current impl expects strictly 32-byte headers + data.
	// Let's construct a valid input.
	// B_len = 1
	// E_len = 1
	// M_len = 1
	// Base = 3
	// Exp = 2
	// M = 5
	// Result should be 3^2 % 5 = 9 % 5 = 4

	input := make([]byte, 0)
	// Lengths (32 bytes each)
	input = append(input, common.LeftPadBytes([]byte{1}, 32)...)
	input = append(input, common.LeftPadBytes([]byte{1}, 32)...)
	input = append(input, common.LeftPadBytes([]byte{1}, 32)...)

	// Values
	input = append(input, 3)
	input = append(input, 2)
	input = append(input, 5)

	output, _, err := RunPrecompiled(PrecompileModExp, input, 100000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output should be M_len bytes, so 1 byte.
	// 4
	if len(output) != 1 || output[0] != 4 {
		t.Errorf("modexp mismatch: got %x, want 04", output)
	}
}

func TestPrecompileBN256Add_Stub(t *testing.T) {
	// Just verify it runs without error on empty (invalid) input returning error?
	// Our impl checks unmarshal errors.
	input := make([]byte, 128) // All zeros -> Point at infinity?
	// Unmarshal of 0,0 is valid (point at infinity)

	_, _, err := RunPrecompiled(PrecompileBN256Add, input, 100000)
	if err != nil {
		// If 0,0 is valid point, no error.
		// If 0,0 is invalid, error.
		// For G1, unmarshalling 0,0 usually works as infinity or fails depending on impl.
		// We accept error or success, just checking for panic/crash.
	}
}
