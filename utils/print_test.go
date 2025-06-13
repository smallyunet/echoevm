package utils

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func TestPrintBytecode(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	PrintBytecode(log, []byte{core.PUSH1, 0x01, core.ADD}, zerolog.InfoLevel)
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("PUSH1")) || !bytes.Contains([]byte(out), []byte("ADD")) {
		t.Fatalf("output missing opcodes: %s", out)
	}
}
