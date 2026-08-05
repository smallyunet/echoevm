// Package trace turns the VM's low-level pre/post hook into stable,
// AI-oriented opcode events. The VM hook remains the execution primitive;
// this package owns presentation semantics such as deltas and explanations.
package trace

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
)

const SchemaVersion = "echoevm.trace.v1"

type Document struct {
	Schema    string          `json:"schema"`
	Execution ExecutionResult `json:"execution"`
	Events    []OpcodeEvent   `json:"events"`
}

type ExecutionResult struct {
	Status         string `json:"status"`
	GasLimit       uint64 `json:"gasLimit"`
	GasUsed        uint64 `json:"gasUsed"`
	ReturnData     string `json:"returnData"`
	TotalSteps     int    `json:"totalSteps"`
	MatchedSteps   int    `json:"matchedSteps"`
	EmittedSteps   int    `json:"emittedSteps"`
	Filtered       bool   `json:"filtered,omitempty"`
	Truncated      bool   `json:"truncated"`
	ExecutionError string `json:"error,omitempty"`
}

type OpcodeEvent struct {
	Type        string          `json:"type"`
	Schema      string          `json:"schema"`
	Step        int             `json:"step"`
	Depth       int             `json:"depth"`
	Address     string          `json:"address"`
	PC          uint64          `json:"pc"`
	Opcode      string          `json:"opcode"`
	OpcodeName  string          `json:"opcodeName"`
	Gas         *GasDelta       `json:"gas,omitempty"`
	Stack       *StackDelta     `json:"stack,omitempty"`
	Memory      *MemoryDelta    `json:"memory,omitempty"`
	Storage     []StorageAccess `json:"storage,omitempty"`
	Control     *ControlFlow    `json:"control,omitempty"`
	Halt        bool            `json:"halt,omitempty"`
	Reverted    bool            `json:"reverted,omitempty"`
	Error       string          `json:"error,omitempty"`
	Explanation string          `json:"explanation,omitempty"`
}

type GasDelta struct {
	Before      uint64  `json:"before"`
	After       uint64  `json:"after"`
	Used        uint64  `json:"used"`
	StaticCost  uint64  `json:"staticCost"`
	DynamicCost *uint64 `json:"dynamicCost,omitempty"`
}

type StackDelta struct {
	SizeBefore int      `json:"sizeBefore"`
	SizeAfter  int      `json:"sizeAfter"`
	Popped     []string `json:"popped,omitempty"`
	Pushed     []string `json:"pushed,omitempty"`
	Reordered  bool     `json:"reordered,omitempty"`
	TopBefore  []string `json:"topBefore,omitempty"`
	TopAfter   []string `json:"topAfter,omitempty"`
}

type MemoryDelta struct {
	SizeBefore int           `json:"sizeBefore"`
	SizeAfter  int           `json:"sizeAfter"`
	Ranges     []MemoryRange `json:"ranges,omitempty"`
	Truncated  bool          `json:"truncated,omitempty"`
}

type MemoryRange struct {
	Offset int    `json:"offset"`
	Before string `json:"before"`
	After  string `json:"after"`
}

type StorageAccess struct {
	Kind           string `json:"kind"`
	Address        string `json:"address"`
	Slot           string `json:"slot"`
	Before         string `json:"before"`
	After          string `json:"after"`
	Original       string `json:"original,omitempty"`
	Warm           bool   `json:"warm"`
	Transient      bool   `json:"transient,omitempty"`
	AppliedInFrame *bool  `json:"appliedInFrame,omitempty"`
}

type ControlFlow struct {
	Kind   string  `json:"kind"`
	Target *uint64 `json:"target,omitempty"`
	Taken  *bool   `json:"taken,omitempty"`
}

type pendingStep struct {
	step int
	pre  vm.TraceStep
}

// Collector pairs nested pre/post hook values without changing execution.
// Events are sorted into pre-op execution order when Events is called.
type Collector struct {
	pending        map[int]pendingStep
	events         []OpcodeEvent
	nextStep       int
	maxMemoryBytes int
}

func NewCollector(maxMemoryBytes int) *Collector {
	if maxMemoryBytes < 0 {
		maxMemoryBytes = 0
	}
	return &Collector{pending: make(map[int]pendingStep), maxMemoryBytes: maxMemoryBytes}
}

func (c *Collector) Consume(raw vm.TraceStep) bool {
	if !raw.IsPost {
		c.pending[raw.Depth] = pendingStep{step: c.nextStep, pre: raw}
		c.nextStep++
		return true
	}
	pending, ok := c.pending[raw.Depth]
	if !ok {
		return true
	}
	c.events = append(c.events, buildEvent(pending.step, pending.pre, &raw, c.maxMemoryBytes))
	delete(c.pending, raw.Depth)
	return true
}

// Events returns completed events plus any opcode whose execution terminated
// before the VM could emit a post state (for example, a recovered panic).
func (c *Collector) Events() []OpcodeEvent {
	for depth, pending := range c.pending {
		c.events = append(c.events, buildEvent(pending.step, pending.pre, nil, c.maxMemoryBytes))
		delete(c.pending, depth)
	}
	sort.Slice(c.events, func(i, j int) bool { return c.events[i].Step < c.events[j].Step })
	return append([]OpcodeEvent(nil), c.events...)
}

func buildEvent(step int, pre vm.TraceStep, post *vm.TraceStep, maxMemoryBytes int) OpcodeEvent {
	afterGas := pre.Gas
	afterStack := pre.Stack
	afterMemory := pre.Memory
	event := OpcodeEvent{
		Type: "opcode", Schema: SchemaVersion, Step: step, Depth: pre.Depth,
		Address: pre.Address, PC: pre.PC, Opcode: fmt.Sprintf("0x%02x", pre.Opcode),
		OpcodeName: pre.OpcodeName,
	}
	if post == nil {
		event.Halt = true
		event.Error = "execution ended before post-op state"
	} else {
		afterGas = post.Gas
		afterStack = post.Stack
		afterMemory = post.Memory
		event.Halt = post.Halt
		event.Reverted = post.Reverted
		event.Error = post.Error
	}
	used := uint64(0)
	if pre.Gas >= afterGas {
		used = pre.Gas - afterGas
	}
	staticCost := core.GasTable[pre.Opcode]
	gas := &GasDelta{Before: pre.Gas, After: afterGas, Used: used, StaticCost: staticCost}
	if used >= staticCost {
		dynamic := used - staticCost
		gas.DynamicCost = &dynamic
	}
	event.Gas = gas
	event.Stack = stackDelta(pre.Opcode, pre.Stack, afterStack)
	event.Memory = memoryDelta(pre.Memory, afterMemory, maxMemoryBytes)
	event.Storage = storageAccesses(pre.Storage, post)
	event.Control = controlFlow(pre, post)
	event.Explanation = explain(event)
	return event
}

func stackDelta(op byte, before, after []string) *StackDelta {
	if op >= core.SWAP1 && op <= core.SWAP1+15 && len(before) == len(after) {
		width := int(op-core.SWAP1) + 2
		return &StackDelta{
			SizeBefore: len(before), SizeAfter: len(after), Reordered: true,
			TopBefore: topWords(before, width), TopAfter: topWords(after, width),
		}
	}
	commonPrefix := 0
	for commonPrefix < len(before) && commonPrefix < len(after) && before[commonPrefix] == after[commonPrefix] {
		commonPrefix++
	}
	delta := &StackDelta{SizeBefore: len(before), SizeAfter: len(after)}
	for index := len(before) - 1; index >= commonPrefix; index-- {
		delta.Popped = append(delta.Popped, before[index])
	}
	delta.Pushed = append(delta.Pushed, after[commonPrefix:]...)
	return delta
}

func topWords(stack []string, count int) []string {
	if count > len(stack) {
		count = len(stack)
	}
	result := make([]string, 0, count)
	for index := len(stack) - 1; index >= len(stack)-count; index-- {
		result = append(result, stack[index])
	}
	return result
}

func memoryDelta(before, after []byte, maxBytes int) *MemoryDelta {
	if len(before) == len(after) {
		equal := true
		for index := range before {
			if before[index] != after[index] {
				equal = false
				break
			}
		}
		if equal {
			return nil
		}
	}
	delta := &MemoryDelta{SizeBefore: len(before), SizeAfter: len(after)}
	maxLen := max(len(before), len(after))
	consumed := 0
	for index := 0; index < maxLen; {
		beforeByte, afterByte := byteAt(before, index), byteAt(after, index)
		if beforeByte == afterByte {
			index++
			continue
		}
		start := index
		var oldBytes, newBytes []byte
		for index < maxLen && byteAt(before, index) != byteAt(after, index) {
			if maxBytes == 0 || consumed >= maxBytes {
				delta.Truncated = true
				break
			}
			oldBytes = append(oldBytes, byteAt(before, index))
			newBytes = append(newBytes, byteAt(after, index))
			consumed++
			index++
		}
		if len(oldBytes) > 0 {
			delta.Ranges = append(delta.Ranges, MemoryRange{
				Offset: start, Before: "0x" + hex.EncodeToString(oldBytes), After: "0x" + hex.EncodeToString(newBytes),
			})
		}
		if delta.Truncated {
			break
		}
	}
	return delta
}

func byteAt(value []byte, index int) byte {
	if index >= len(value) {
		return 0
	}
	return value[index]
}

func storageAccesses(raw []vm.TraceStorageAccess, post *vm.TraceStep) []StorageAccess {
	if len(raw) == 0 {
		return nil
	}
	result := make([]StorageAccess, 0, len(raw))
	for _, item := range raw {
		access := StorageAccess{
			Kind: item.Kind, Address: item.Address, Slot: item.Slot, Before: item.Before,
			After: item.After, Original: item.Original, Warm: item.Warm, Transient: item.Transient,
		}
		if item.Kind == "write" {
			applied := post != nil && post.Error == ""
			access.AppliedInFrame = &applied
		}
		result = append(result, access)
	}
	return result
}

func controlFlow(pre vm.TraceStep, post *vm.TraceStep) *ControlFlow {
	if post == nil {
		return nil
	}
	switch pre.Opcode {
	case core.JUMP, core.JUMPI:
		flow := &ControlFlow{Kind: "jump"}
		if len(pre.Stack) > 0 {
			if target, err := strconv.ParseUint(strings.TrimPrefix(pre.Stack[len(pre.Stack)-1], "0x"), 16, 64); err == nil {
				flow.Target = &target
			}
		}
		taken := post.PC != pre.PC+1
		flow.Taken = &taken
		return flow
	case core.CALL, core.CALLCODE, core.DELEGATECALL, core.STATICCALL:
		return &ControlFlow{Kind: "call"}
	case core.CREATE, core.CREATE2:
		return &ControlFlow{Kind: "create"}
	case core.RETURN:
		return &ControlFlow{Kind: "return"}
	case core.REVERT:
		return &ControlFlow{Kind: "revert"}
	case core.STOP:
		return &ControlFlow{Kind: "stop"}
	default:
		return nil
	}
}

func explain(event OpcodeEvent) string {
	if event.Error != "" {
		return fmt.Sprintf("%s halted: %s", event.OpcodeName, event.Error)
	}
	if len(event.Storage) > 0 {
		access := event.Storage[0]
		location := "persistent storage"
		if access.Transient {
			location = "transient storage"
		}
		if access.Kind == "read" {
			return fmt.Sprintf("Read %s slot %s as %s (%s access).", location, access.Slot, access.After, warmth(access.Warm))
		}
		return fmt.Sprintf("Write %s slot %s from %s to %s (%s access).", location, access.Slot, access.Before, access.After, warmth(access.Warm))
	}
	if event.Control != nil {
		switch event.Control.Kind {
		case "jump":
			if event.Control.Target != nil && event.Control.Taken != nil {
				return fmt.Sprintf("Jump target is PC %d; taken=%t.", *event.Control.Target, *event.Control.Taken)
			}
		case "call":
			return "Enter a contract call frame and return its success flag."
		case "create":
			return "Execute contract initcode and return the created address or zero."
		case "return":
			return "Halt this call frame successfully and return a memory slice."
		case "revert":
			return "Revert this call frame and return a memory slice as error data."
		case "stop":
			return "Halt this call frame successfully."
		}
	}
	if event.Stack != nil {
		if event.Stack.Reordered {
			return fmt.Sprintf("Reorder the stack top from %v to %v.", event.Stack.TopBefore, event.Stack.TopAfter)
		}
		if len(event.Stack.Popped) > 0 || len(event.Stack.Pushed) > 0 {
			return fmt.Sprintf("Pop %d stack word(s), push %d word(s).", len(event.Stack.Popped), len(event.Stack.Pushed))
		}
	}
	if event.Memory != nil {
		return fmt.Sprintf("Resize memory from %d to %d bytes.", event.Memory.SizeBefore, event.Memory.SizeAfter)
	}
	return fmt.Sprintf("Execute %s without a visible stack, memory, or storage change.", event.OpcodeName)
}

func warmth(warm bool) string {
	if warm {
		return "warm"
	}
	return "cold"
}
