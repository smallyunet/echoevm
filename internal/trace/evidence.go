package trace

import (
	"fmt"
	"sort"
	"strings"

	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

const EvidenceSchemaVersion = "echoevm.evidence.v1"

const (
	ProfileAuto    = "auto"
	ProfileRevert  = "revert"
	ProfileStorage = "storage"
	ProfileCall    = "call"
	ProfileABI     = "abi"
	ProfileGas     = "gas"
	ProfileMath    = "arithmetic"
	ProfileFull    = "full"
)

type EvidenceDocument struct {
	Schema    string            `json:"schema"`
	Profile   string            `json:"profile"`
	Execution EvidenceExecution `json:"execution"`
	Events    []EvidenceEvent   `json:"events"`
	Links     []EvidenceLink    `json:"links,omitempty"`
	Selection EvidenceSelection `json:"selection"`
}

type EvidenceLink struct {
	Kind  string           `json:"kind"`
	From  EvidenceLocation `json:"from"`
	To    EvidenceLocation `json:"to"`
	Input int              `json:"input,omitempty"`
	Value string           `json:"value,omitempty"`
}

type EvidenceLocation struct {
	Step  int    `json:"step"`
	Depth int    `json:"depth,omitempty"`
	PC    uint64 `json:"pc"`
	Op    string `json:"op"`
}

type EvidenceExecution struct {
	Status     string `json:"status"`
	GasUsed    uint64 `json:"gasUsed"`
	ReturnData string `json:"returnData"`
	TotalSteps int    `json:"totalSteps"`
	Error      string `json:"error,omitempty"`
}

type EvidenceSelection struct {
	Candidates int  `json:"candidates"`
	Selected   int  `json:"selected"`
	Omitted    int  `json:"omitted"`
	Truncated  bool `json:"truncated"`
}

type EvidenceEvent struct {
	Step        int                     `json:"step"`
	Depth       int                     `json:"depth,omitempty"`
	Address     string                  `json:"address,omitempty"`
	PC          uint64                  `json:"pc"`
	Op          string                  `json:"op"`
	Gas         *GasDelta               `json:"gas,omitempty"`
	Stack       *EvidenceStack          `json:"stack,omitempty"`
	Memory      *MemoryDelta            `json:"memory,omitempty"`
	Storage     []EvidenceStorageAccess `json:"storage,omitempty"`
	Control     *ControlFlow            `json:"control,omitempty"`
	Halt        bool                    `json:"halt,omitempty"`
	Reverted    bool                    `json:"reverted,omitempty"`
	Error       string                  `json:"error,omitempty"`
	Explanation string                  `json:"why,omitempty"`
}

type EvidenceStorageAccess struct {
	Kind           string `json:"kind"`
	Address        string `json:"address,omitempty"`
	Slot           string `json:"slot"`
	Before         string `json:"before"`
	After          string `json:"after"`
	Original       string `json:"original,omitempty"`
	Warm           bool   `json:"warm,omitempty"`
	Transient      bool   `json:"transient,omitempty"`
	AppliedInFrame *bool  `json:"appliedInFrame,omitempty"`
}

type EvidenceStack struct {
	Popped    []string `json:"pop,omitempty"`
	Pushed    []string `json:"push,omitempty"`
	Reordered bool     `json:"reordered,omitempty"`
	TopBefore []string `json:"before,omitempty"`
	TopAfter  []string `json:"after,omitempty"`
}

type evidenceCandidate struct {
	priority int
	event    OpcodeEvent
}

func ValidateEvidenceProfile(profile string) error {
	switch profile {
	case ProfileAuto, ProfileRevert, ProfileStorage, ProfileCall, ProfileABI, ProfileGas, ProfileMath, ProfileFull:
		return nil
	default:
		return fmt.Errorf("unsupported evidence profile %q (want auto, revert, storage, call, abi, gas, arithmetic, or full)", profile)
	}
}

// BuildEvidence turns an already-filtered execution trace into a deterministic,
// bounded causal view. The event limit affects presentation only; execution and
// collection have already completed.
func BuildEvidence(execution ExecutionResult, events []OpcodeEvent, profile string, limit int) (EvidenceDocument, error) {
	if err := ValidateEvidenceProfile(profile); err != nil {
		return EvidenceDocument{}, err
	}
	candidates := make([]evidenceCandidate, 0, len(events))
	for _, event := range events {
		if !profileSelects(profile, event) {
			continue
		}
		candidates = append(candidates, evidenceCandidate{priority: evidencePriority(profile, event), event: event})
	}
	selected := append([]evidenceCandidate(nil), candidates...)
	truncated := limit > 0 && len(selected) > limit
	if truncated {
		sort.SliceStable(selected, func(i, j int) bool {
			if selected[i].priority == selected[j].priority {
				return selected[i].event.Step < selected[j].event.Step
			}
			return selected[i].priority > selected[j].priority
		})
		selected = selected[:limit]
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].event.Step < selected[j].event.Step })

	evidenceEvents := make([]EvidenceEvent, 0, len(selected))
	selectedEvents := make([]OpcodeEvent, 0, len(selected))
	for _, candidate := range selected {
		selectedEvents = append(selectedEvents, candidate.event)
		evidenceEvents = append(evidenceEvents, compactEvidenceEvent(candidate.event, profile))
	}
	return EvidenceDocument{
		Schema:  EvidenceSchemaVersion,
		Profile: profile,
		Execution: EvidenceExecution{
			Status: execution.Status, GasUsed: execution.GasUsed, ReturnData: execution.ReturnData,
			TotalSteps: execution.TotalSteps, Error: execution.ExecutionError,
		},
		Events: evidenceEvents,
		Links:  buildEvidenceLinks(events, selectedEvents),
		Selection: EvidenceSelection{
			Candidates: len(candidates), Selected: len(evidenceEvents), Omitted: len(events) - len(evidenceEvents), Truncated: truncated,
		},
	}, nil
}

func buildEvidenceLinks(allEvents, events []OpcodeEvent) []EvidenceLink {
	links := buildValueFlowLinks(allEvents, events)
	for index, event := range events {
		if !controlKind(event, "call", "create") {
			continue
		}
		firstChild, lastChild := -1, -1
		for next := index + 1; next < len(events); next++ {
			if events[next].Depth <= event.Depth {
				break
			}
			if events[next].Depth == event.Depth+1 {
				if firstChild == -1 {
					firstChild = next
				}
				lastChild = next
			}
		}
		if firstChild == -1 {
			continue
		}
		links = append(links,
			EvidenceLink{Kind: "enters-frame", From: evidenceLocation(event), To: evidenceLocation(events[firstChild])},
			EvidenceLink{Kind: "returns-to", From: evidenceLocation(events[lastChild]), To: evidenceLocation(event)},
		)
	}
	for index, terminal := range events {
		if !terminalRollsBack(terminal) {
			continue
		}
		frameStart := -1
		if terminal.Depth > 0 {
			for prior := index - 1; prior >= 0; prior-- {
				if events[prior].Depth == terminal.Depth-1 && controlKind(events[prior], "call", "create") {
					frameStart = prior
					break
				}
			}
		}
		for prior := frameStart + 1; prior < index; prior++ {
			if events[prior].Depth != terminal.Depth || !hasStorageWrite(events[prior]) {
				continue
			}
			links = append(links, EvidenceLink{
				Kind: "rolls-back", From: evidenceLocation(events[prior]), To: evidenceLocation(terminal),
			})
		}
	}
	return links
}

type stackOrigin struct {
	location EvidenceLocation
	known    bool
}

func buildValueFlowLinks(allEvents, selected []OpcodeEvent) []EvidenceLink {
	selectedSteps := make(map[int]struct{}, len(selected))
	for _, event := range selected {
		selectedSteps[event.Step] = struct{}{}
	}
	stacks := make(map[int][]stackOrigin)
	links := make([]EvidenceLink, 0)
	previousDepth := 0
	for index, event := range allEvents {
		if index == 0 || event.Depth > previousDepth {
			stacks[event.Depth] = nil
		}
		previousDepth = event.Depth
		if event.Stack == nil {
			continue
		}
		stack := stacks[event.Depth]
		if len(stack) < event.Stack.SizeBefore {
			stack = append(make([]stackOrigin, event.Stack.SizeBefore-len(stack)), stack...)
		} else if len(stack) > event.Stack.SizeBefore {
			stack = stack[len(stack)-event.Stack.SizeBefore:]
		}
		if _, consumerSelected := selectedSteps[event.Step]; consumerSelected {
			for input, value := range event.Stack.Popped {
				originIndex := len(stack) - 1 - input
				if originIndex < 0 || !stack[originIndex].known {
					continue
				}
				origin := stack[originIndex].location
				if _, producerSelected := selectedSteps[origin.Step]; !producerSelected || origin.Step == event.Step {
					continue
				}
				links = append(links, EvidenceLink{
					Kind: "value-flow", From: origin, To: evidenceLocation(event), Input: input, Value: compactWord(value),
				})
			}
		}
		op := opcodeByte(event)
		switch {
		case op >= core.SWAP1 && op <= core.SWAP1+15:
			width := int(op-core.SWAP1) + 2
			if len(stack) >= width {
				top, other := len(stack)-1, len(stack)-width
				stack[top], stack[other] = stack[other], stack[top]
			}
		case op >= core.DUP1 && op <= core.DUP1+15:
			depth := int(op-core.DUP1) + 1
			origin := stackOrigin{}
			if len(stack) >= depth {
				origin = stack[len(stack)-depth]
			}
			stack = append(stack, origin)
		default:
			popped := len(event.Stack.Popped)
			if popped > len(stack) {
				popped = len(stack)
			}
			stack = stack[:len(stack)-popped]
			origin := stackOrigin{location: evidenceLocation(event), known: true}
			for range event.Stack.Pushed {
				stack = append(stack, origin)
			}
		}
		stacks[event.Depth] = stack
	}
	return links
}

func evidenceLocation(event OpcodeEvent) EvidenceLocation {
	return EvidenceLocation{Step: event.Step, Depth: event.Depth, PC: event.PC, Op: event.OpcodeName}
}

func terminalRollsBack(event OpcodeEvent) bool {
	return event.Reverted || event.Error != "" || controlKind(event, "revert")
}

func hasStorageWrite(event OpcodeEvent) bool {
	for _, access := range event.Storage {
		if access.Kind == "write" {
			return true
		}
	}
	return false
}

func profileSelects(profile string, event OpcodeEvent) bool {
	if profile == ProfileFull {
		return true
	}
	if event.Halt || event.Reverted || event.Error != "" {
		return true
	}
	switch profile {
	case ProfileAuto, ProfileGas:
		return !isStackPlumbing(event)
	case ProfileRevert:
		return event.Memory != nil || len(event.Storage) > 0 || controlKind(event, "call", "create", "revert", "return") || isReturnDataOpcode(event)
	case ProfileStorage:
		return len(event.Storage) > 0 || controlKind(event, "call", "create", "revert", "return")
	case ProfileCall:
		return controlKind(event, "call", "create", "revert", "return") || (event.Depth > 0 && !isStackPlumbing(event))
	case ProfileABI:
		return event.Memory != nil || controlKind(event, "return", "revert") || isABIOpcode(event)
	case ProfileMath:
		return event.Memory != nil || controlKind(event, "return", "revert") || isArithmeticOpcode(event)
	default:
		return false
	}
}

func evidencePriority(profile string, event OpcodeEvent) int {
	if event.Error != "" || event.Halt || event.Reverted {
		return 1000
	}
	if profile == ProfileMath {
		switch opcodeByte(event) {
		case core.DIV, core.SDIV, core.MOD, core.SMOD, core.ADDMOD, core.MULMOD:
			return 950
		case core.SUB:
			return 900
		}
		if isArithmeticOpcode(event) {
			return 800
		}
	}
	if len(event.Storage) > 0 {
		return 900
	}
	if controlKind(event, "revert") {
		return 850
	}
	if controlKind(event, "call", "create") {
		return 800
	}
	if controlKind(event, "return") {
		return 750
	}
	if event.Memory != nil {
		return 700
	}
	if event.Depth > 0 {
		return 650
	}
	return 500
}

func compactEvidenceEvent(event OpcodeEvent, profile string) EvidenceEvent {
	address := event.Address
	if address == "" || common.HexToAddress(address) == (common.Address{}) {
		address = ""
	}
	result := EvidenceEvent{
		Step: event.Step, Depth: event.Depth, Address: address, PC: event.PC, Op: event.OpcodeName,
		Memory: event.Memory, Storage: compactStorage(event.Storage, address), Control: event.Control,
		Halt: event.Halt, Reverted: event.Reverted, Error: event.Error,
	}
	if event.Error != "" {
		result.Explanation = event.Explanation
	}
	if profile == ProfileGas || profile == ProfileFull {
		result.Gas = event.Gas
	}
	if event.Stack != nil && (len(event.Stack.Popped) > 0 || len(event.Stack.Pushed) > 0 || event.Stack.Reordered) {
		result.Stack = &EvidenceStack{
			Popped: event.Stack.Popped, Pushed: event.Stack.Pushed, Reordered: event.Stack.Reordered,
			TopBefore: event.Stack.TopBefore, TopAfter: event.Stack.TopAfter,
		}
	}
	return result
}

func compactStorage(items []StorageAccess, eventAddress string) []EvidenceStorageAccess {
	if len(items) == 0 {
		return nil
	}
	result := make([]EvidenceStorageAccess, 0, len(items))
	for _, item := range items {
		address := item.Address
		if address == eventAddress || common.HexToAddress(address) == (common.Address{}) {
			address = ""
		}
		original := ""
		if item.Original != "" {
			original = compactWord(item.Original)
		}
		before := compactWord(item.Before)
		if original == before {
			original = ""
		}
		applied := item.AppliedInFrame
		if applied != nil && *applied {
			applied = nil
		}
		result = append(result, EvidenceStorageAccess{
			Kind: item.Kind, Address: address, Slot: compactWord(item.Slot), Before: before,
			After: compactWord(item.After), Original: original, Warm: item.Warm,
			Transient: item.Transient, AppliedInFrame: applied,
		})
	}
	return result
}

func compactWord(value string) string {
	value = strings.TrimPrefix(strings.ToLower(value), "0x")
	value = strings.TrimLeft(value, "0")
	if value == "" {
		value = "0"
	}
	return "0x" + value
}

func isStackPlumbing(event OpcodeEvent) bool {
	op := opcodeByte(event)
	return op == core.PUSH0 || (op >= core.PUSH1 && op <= core.PUSH1+31) ||
		(op >= core.DUP1 && op <= core.DUP1+15) || (op >= core.SWAP1 && op <= core.SWAP1+15) ||
		op == core.POP || op == core.JUMPDEST
}

func opcodeByte(event OpcodeEvent) byte {
	op := byte(0)
	_, _ = fmt.Sscanf(event.Opcode, "0x%02x", &op)
	return op
}

func controlKind(event OpcodeEvent, kinds ...string) bool {
	if event.Control == nil {
		return false
	}
	for _, kind := range kinds {
		if event.Control.Kind == kind {
			return true
		}
	}
	return false
}

func isABIOpcode(event OpcodeEvent) bool {
	switch event.OpcodeName {
	case "CALLDATALOAD", "CALLDATASIZE", "CALLDATACOPY", "MLOAD", "MSTORE", "MSTORE8", "MCOPY", "SHA3", "KECCAK256", "RETURNDATASIZE", "RETURNDATACOPY":
		return true
	default:
		return false
	}
}

func isReturnDataOpcode(event OpcodeEvent) bool {
	return event.OpcodeName == "RETURNDATASIZE" || event.OpcodeName == "RETURNDATACOPY"
}

func isArithmeticOpcode(event OpcodeEvent) bool {
	op := opcodeByte(event)
	return (op >= core.ADD && op <= core.SIGNEXTEND) ||
		(op >= core.LT && op <= core.SAR) || op == core.SHA3
}
