package trace

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
)

func TestCollectorBuildsStackGasAndMemoryDeltas(t *testing.T) {
	collector := NewCollector(256)
	collector.Consume(vm.TraceStep{
		PC: 4, Opcode: core.MSTORE8, OpcodeName: "MSTORE8", Stack: []string{"0xaa", "0x0"},
		Gas: 100, Memory: nil, Address: common.Address{}.Hex(),
	})
	collector.Consume(vm.TraceStep{
		PC: 5, Opcode: core.MSTORE8, OpcodeName: "MSTORE8", Stack: []string{},
		Gas: 94, Memory: append([]byte{0xaa}, make([]byte, 31)...), Address: common.Address{}.Hex(), IsPost: true,
	})

	events := collector.Events()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	event := events[0]
	if event.Gas.Used != 6 || event.Gas.StaticCost != core.GasTable[core.MSTORE8] || event.Gas.DynamicCost == nil || *event.Gas.DynamicCost != 3 {
		t.Fatalf("gas delta = %+v", event.Gas)
	}
	if len(event.Stack.Popped) != 2 || event.Stack.Popped[0] != "0x0" || event.Stack.Popped[1] != "0xaa" {
		t.Fatalf("stack delta = %+v", event.Stack)
	}
	if event.Memory == nil || event.Memory.SizeAfter != 32 || len(event.Memory.Ranges) != 1 || event.Memory.Ranges[0].After != "0xaa" {
		t.Fatalf("memory delta = %+v", event.Memory)
	}
}

func TestCollectorExplainsPersistentStorage(t *testing.T) {
	address := common.HexToAddress("0x1000000000000000000000000000000000000001")
	state := core.NewMemoryStateDB()
	state.PrepareTransaction()
	code := []byte{
		core.PUSH1, 0x2a, core.PUSH1, 0x01, core.SSTORE,
		core.PUSH1, 0x01, core.SLOAD, core.STOP,
	}
	intr := vm.New(code, state, address)
	intr.SetGas(100_000)
	intr.SetTraceDetails(true)
	collector := NewCollector(256)
	intr.RunWithHook(collector.Consume)
	if intr.Err() != nil {
		t.Fatal(intr.Err())
	}

	var store, load *OpcodeEvent
	events := collector.Events()
	for index := range events {
		event := events[index]
		switch event.OpcodeName {
		case "SSTORE":
			store = &event
		case "SLOAD":
			load = &event
		}
	}
	if store == nil || len(store.Storage) != 1 || store.Storage[0].AppliedInFrame == nil || !*store.Storage[0].AppliedInFrame {
		t.Fatalf("SSTORE event = %+v", store)
	}
	if !strings.HasSuffix(store.Storage[0].After, "2a") || !strings.Contains(store.Explanation, "Write persistent storage") {
		t.Fatalf("SSTORE explanation = %+v", store)
	}
	if load == nil || len(load.Storage) != 1 || !load.Storage[0].Warm || !strings.Contains(load.Explanation, "Read persistent storage") {
		t.Fatalf("SLOAD event = %+v", load)
	}
}

func TestCollectorPreservesNestedPreOrder(t *testing.T) {
	collector := NewCollector(32)
	collector.Consume(vm.TraceStep{PC: 0, Opcode: core.CALL, OpcodeName: "CALL", Gas: 100, Depth: 0})
	collector.Consume(vm.TraceStep{PC: 0, Opcode: core.STOP, OpcodeName: "STOP", Gas: 50, Depth: 1})
	collector.Consume(vm.TraceStep{PC: 1, Opcode: core.STOP, OpcodeName: "STOP", Gas: 50, Depth: 1, IsPost: true, Halt: true})
	collector.Consume(vm.TraceStep{PC: 1, Opcode: core.CALL, OpcodeName: "CALL", Gas: 80, Depth: 0, IsPost: true})
	events := collector.Events()
	if len(events) != 2 || events[0].Step != 0 || events[0].Depth != 0 || events[1].Step != 1 || events[1].Depth != 1 {
		t.Fatalf("nested events = %+v", events)
	}
}

func TestCollectorDescribesSwapAsReordering(t *testing.T) {
	collector := NewCollector(32)
	collector.Consume(vm.TraceStep{PC: 0, Opcode: core.SWAP1, OpcodeName: "SWAP1", Gas: 10, Stack: []string{"0x1", "0x2"}})
	collector.Consume(vm.TraceStep{PC: 1, Opcode: core.SWAP1, OpcodeName: "SWAP1", Gas: 7, Stack: []string{"0x2", "0x1"}, IsPost: true})
	event := collector.Events()[0]
	if !event.Stack.Reordered || len(event.Stack.Popped) != 0 || event.Stack.TopBefore[0] != "0x2" || event.Stack.TopAfter[0] != "0x1" {
		t.Fatalf("SWAP delta = %+v", event.Stack)
	}
}
