package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestE2E_Run(t *testing.T) {
	// Build the binary first
	cmd := exec.Command("go", "build", "-o", "echoevm_test", "../../cmd/echoevm")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build echoevm: %v", err)
	}
	defer func() { _ = os.Remove("echoevm_test") }()

	binPath, _ := filepath.Abs("echoevm_test")
	tempDir := t.TempDir()
	prestatePath := filepath.Join(tempDir, "prestate.json")
	prestate := []byte(`{
  "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa": {
    "balance": "0x100000000",
    "nonce": "0x0",
    "code": "0x",
    "storage": {}
  },
  "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb": {
    "balance": "0x0",
    "nonce": "0x0",
    "code": "0xfe",
    "storage": {}
  },
  "0xcccccccccccccccccccccccccccccccccccccccc": {
    "balance": "0x0",
    "nonce": "0x0",
    "code": "0x60006000fd",
    "storage": {}
  }
}`)
	if err := os.WriteFile(prestatePath, prestate, 0o600); err != nil {
		t.Fatal(err)
	}
	writeTransaction := func(name, to string) string {
		t.Helper()
		path := filepath.Join(tempDir, name)
		contents := []byte(`{
  "to": "` + to + `",
  "data": "0x",
  "value": "0x0",
  "gasLimit": "0xc350",
  "gasPrice": "0x1",
  "sender": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "nonce": "0x0"
}`)
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	invalidTransactionPath := writeTransaction("invalid.json", "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	revertTransactionPath := writeTransaction("revert.json", "0xcccccccccccccccccccccccccccccccccccccccc")

	tests := []struct {
		name     string
		args     []string
		wantOut  string // simple substring match
		wantCode int
	}{
		{
			name:     "version",
			args:     []string{"version"},
			wantOut:  "echoevm v0.0.21",
			wantCode: 0,
		},
		{
			name:     "simple add",
			args:     []string{"run", "6001600201"}, // PUSH1 1 PUSH1 2 ADD
			wantOut:  "Return: 0x",
			wantCode: 0,
		},
		{
			name:     "simple add with debug",
			args:     []string{"run", "--debug", "6001600201"},
			wantOut:  "ADD",
			wantCode: 0,
		},
		{
			name:     "invalid opcode",
			args:     []string{"run", "fe"},
			wantOut:  "execution failed: invalid opcode: 0xfe",
			wantCode: 1,
		},
		{
			name:     "transaction invalid opcode returns JSON and failure",
			args:     []string{"run", "--prestate", prestatePath, "--tx", invalidTransactionPath},
			wantOut:  `"error": "invalid opcode: 0xfe"`,
			wantCode: 1,
		},
		{
			name:     "transaction revert returns JSON and failure",
			args:     []string{"run", "--prestate", prestatePath, "--tx", revertTransactionPath},
			wantOut:  `"reverted": true`,
			wantCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := exec.Command(binPath, tt.args...)
			var out bytes.Buffer
			ps.Stdout = &out
			ps.Stderr = &out

			err := ps.Run()

			// Check exit code
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					t.Fatalf("failed to run echoevm_test: %v", err)
				}
			}

			if exitCode != tt.wantCode {
				t.Errorf("expected exit code %d, got %d. Output: %s", tt.wantCode, exitCode, out.String())
			}

			if tt.wantOut != "" && !bytes.Contains(out.Bytes(), []byte(tt.wantOut)) {
				t.Errorf("expected output to contain %q, got %q", tt.wantOut, out.String())
			}
		})
	}
}
