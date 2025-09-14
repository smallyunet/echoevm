package vm

import (
	"encoding/hex"
	"math/big"
	"testing"
)

// helper to create 32-byte big int from small int
func bi(v int64) *big.Int { return big.NewInt(v) }

func TestLog0(t *testing.T) {
	i := New([]byte{})
	// Prepare memory: write 4 bytes "deadbeef"
	data, _ := hex.DecodeString("deadbeef")
	i.memory.Write(0, data)
	// Stack per our implementation pop order: size, offset
	i.stack.PushSafe(big.NewInt(int64(0))) // (no topics)
	// Actually for LOG0: we expect push order: offset, size so that Pop gives size then offset.
	// Push offset then size:
	i.stack.PushSafe(big.NewInt(0)) // offset
	i.stack.PushSafe(big.NewInt(4)) // size
	opLog(i, 0xa0)
	logs := i.Logs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log got %d", len(logs))
	}
	if logs[0].Data != "0xdeadbeef" {
		t.Fatalf("unexpected data %s", logs[0].Data)
	}
	if len(logs[0].Topics) != 0 {
		t.Fatalf("expected no topics")
	}
}

func TestLog2(t *testing.T) {
	i := New([]byte{})
	payload := []byte{0xca, 0xfe, 0xba, 0xbe}
	i.memory.Write(16, payload)
	// For LOG2 we need: offset, size, topic0, topic1 (stack pushes offset, size, topic0, topic1) -> pops size, offset, topic0, topic1
	i.stack.PushSafe(big.NewInt(16)) // offset
	i.stack.PushSafe(big.NewInt(4))  // size
	i.stack.PushSafe(bi(1))          // topic0
	i.stack.PushSafe(bi(2))          // topic1
	opLog(i, 0xa2)
	logs := i.Logs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log got %d", len(logs))
	}
	if logs[0].Data != "0xcafebabe" {
		t.Fatalf("unexpected data %s", logs[0].Data)
	}
	if len(logs[0].Topics) != 2 {
		t.Fatalf("expected 2 topics")
	}
}
