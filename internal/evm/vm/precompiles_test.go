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
		{common.BytesToAddress([]byte{0x05}), false},
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
