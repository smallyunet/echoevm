package vm

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

// helper to create 32-byte big int from small int
func bi(v int64) *big.Int { return big.NewInt(v) }

func TestLog0(t *testing.T) {
	i := New([]byte{}, core.NewMemoryStateDB(), common.Address{})
	// Prepare memory: write 4 bytes "deadbeef"
	data, _ := hex.DecodeString("deadbeef")
	i.memory.Write(0, data)
	// Stack per our implementation pop order: size, offset
	// Stack: [offset, size] (offset on top)
	i.stack.PushSafe(big.NewInt(4)) // size
	i.stack.PushSafe(big.NewInt(0)) // offset
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

func TestOpLog(t *testing.T) {
	code := []byte{
		core.PUSH1, 0x01, // topic1
		core.PUSH1, 0x01, // size
		core.PUSH1, 0x00, // offset
		core.LOG1,
	}
	i := New(code, core.NewMemoryStateDB(), common.Address{})
	i.Run()
	if len(i.Logs()) != 1 {
		t.Fatalf("expected 1 log, got %d", len(i.Logs()))
	}
}

func TestOpLogData(t *testing.T) {
	code := []byte{
		core.PUSH1, 0xAA,
		core.PUSH1, 0x00,
		core.MSTORE8,
		core.PUSH1, 0x01,
		core.PUSH1, 0x00,
		core.LOG0,
	}
	i := New(code, core.NewMemoryStateDB(), common.Address{})
	i.Run()
	if len(i.Logs()) != 1 {
		t.Fatalf("expected 1 log, got %d", len(i.Logs()))
	}
	if i.Logs()[0].Data != "0xaa" {
		t.Fatalf("expected log data 0xaa, got %s", i.Logs()[0].Data)
	}
	if len(i.Logs()[0].Topics) != 0 {
		t.Fatalf("expected no topics")
	}
}

func TestLog2(t *testing.T) {
	i := New([]byte{}, core.NewMemoryStateDB(), common.Address{})
	payload := []byte{0xca, 0xfe, 0xba, 0xbe}
	i.memory.Write(16, payload)
	// Stack: [offset, size, topic0, topic1] (offset on top)
	i.stack.PushSafe(bi(2))          // topic1
	i.stack.PushSafe(bi(1))          // topic0
	i.stack.PushSafe(big.NewInt(4))  // size
	i.stack.PushSafe(big.NewInt(16)) // offset
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
