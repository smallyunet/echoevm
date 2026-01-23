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
	defer os.Remove("echoevm_test")

	binPath, _ := filepath.Abs("echoevm_test")

	tests := []struct {
		name     string
		args     []string
		wantOut  string // simple substring match
		wantCode int
	}{
		{
			name:     "version",
			args:     []string{"version"},
			wantOut:  "echoevm v0.0.17",
			wantCode: 0,
		},
		{
			name:     "simple add",
			args:     []string{"run", "6001600201"}, // PUSH1 1 PUSH1 2 ADD
			wantOut:  "",                            // Run doesn't output stack unless debug, wait, it might finish silently
			wantCode: 0,
		},
		{
			name:     "simple add with debug",
			args:     []string{"run", "--debug", "6001600201"},
			wantOut:  "ADD",
			wantCode: 0,
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
