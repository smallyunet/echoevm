package trace

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
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
	ProfileFull    = "full"
)

type EvidenceDocument struct {
	Schema    string            `json:"schema"`
	Profile   string            `json:"profile"`
	Execution EvidenceExecution `json:"execution"`
	Events    []EvidenceEvent   `json:"events"`
	Selection EvidenceSelection `json:"selection"`
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
	case ProfileAuto, ProfileRevert, ProfileStorage, ProfileCall, ProfileABI, ProfileGas, ProfileFull:
		return nil
	default:
		return fmt.Errorf("unsupported evidence profile %q (want auto, revert, storage, call, abi, gas, or full)", profile)
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
		candidates = append(candidates, evidenceCandidate{priority: evidencePriority(event), event: event})
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
	for _, candidate := range selected {
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
		Selection: EvidenceSelection{
			Candidates: len(candidates), Selected: len(evidenceEvents), Omitted: len(events) - len(evidenceEvents), Truncated: truncated,
		},
	}, nil
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
	default:
		return false
	}
}

func evidencePriority(event OpcodeEvent) int {
	if event.Error != "" || event.Halt || event.Reverted {
		return 1000
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
	op := byte(0)
	if _, err := fmt.Sscanf(event.Opcode, "0x%02x", &op); err != nil {
		return false
	}
	return op == core.PUSH0 || (op >= core.PUSH1 && op <= core.PUSH1+31) ||
		(op >= core.DUP1 && op <= core.DUP1+15) || (op >= core.SWAP1 && op <= core.SWAP1+15) ||
		op == core.POP || op == core.JUMPDEST
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
