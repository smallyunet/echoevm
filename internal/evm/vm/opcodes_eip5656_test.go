package vm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func TestOpMcopy(t *testing.T) {
	// Bytecode: PUSH1 0x05 (size) PUSH1 0x00 (src) PUSH1 0x10 (dest) MCOPY
	// Memory before: [0:5] = 0xAA...
	// Memory after: [16:21] = 0xAA...
	
	// We need to setup memory first.
	// PUSH1 0xAA PUSH1 0x00 MSTORE8 ... repeat or just use one word MSTORE
	// Let's use MSTORE to put 0x11223344... at address 0
	// PUSH32 0x11223344... PUSH1 0x00 MSTORE
	// Then MCOPY to 0x40
	
	// Bytecode construction:
	// PUSH32 0x1122...33 PUSH1 0x00 MSTORE
	// PUSH1 0x20 (32 bytes) PUSH1 0x00 (src) PUSH1 0x40 (dest) MCOPY
	
	val := common.HexToHash("0x112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00")
	
	// 7f... (PUSH32)
	code := []byte{0x7f}
	code = append(code, val.Bytes()...)
	code = append(code, []byte{
		byte(core.PUSH1), 0x00,
		byte(core.MSTORE),
		// MCOPY
		byte(core.PUSH1), 0x20, // size 32
		byte(core.PUSH1), 0x00, // src 0
		byte(core.PUSH1), 0x40, // dest 64
		byte(core.MCOPY),
		byte(core.STOP),
	}...)

	memDB := core.NewMemoryStateDB()
	interpreter := New(code, memDB, common.Address{})
	interpreter.SetGas(100000)
	
	interpreter.Run()
	
	if interpreter.Err() != nil {
		t.Fatalf("Execution failed: %v", interpreter.Err())
	}
	
	// Check memory at 0x40
	mem := interpreter.Memory()
	// Get 32 bytes from 0x40
	result := mem.Get(64)
	
	if common.BytesToHash(result) != val {
		t.Errorf("MCOPY failed. Expected %x, got %x", val, result)
	}
}
