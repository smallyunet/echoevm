package differential

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
)

var contractAddress = common.BytesToAddress([]byte("contract"))

type EchoRunner struct{}

func (EchoRunner) Run(ctx context.Context, req Request) (ExecutionResult, error) {
	code, _ := decodeHexField("bytecode", req.Bytecode)
	input, _ := decodeHexField("calldata", req.Calldata)
	state := core.NewMemoryStateDB()
	for key, value := range req.InitialStorage {
		state.InitState(contractAddress, common.HexToHash(key), common.HexToHash(value))
	}
	state.PrepareTransaction()
	state.AddAddressToAccessList(contractAddress)
	intr := vm.NewWithCallData(code, input, state, contractAddress)
	intr.SetGas(req.GasLimit)
	intr.SetBlockGasLimit(req.GasLimit)

	trace := make([]NormalizedStep, 0, 128)
	var pending *NormalizedStep
	var runErr error
	intr.RunWithHook(func(raw vm.TraceStep) bool {
		if err := ctx.Err(); err != nil {
			runErr = err
			return false
		}
		if !raw.IsPost {
			if len(trace) >= MaxTraceSteps {
				runErr = fmt.Errorf("trace exceeds maximum %d steps", MaxTraceSteps)
				return false
			}
			step := NormalizedStep{
				Index: len(trace), Depth: 0, PC: raw.PC,
				Opcode: fmt.Sprintf("0x%02x", raw.Opcode), OpcodeName: gethvm.OpCode(raw.Opcode).String(),
				GasBefore: raw.Gas, StackBefore: canonicalStack(raw.Stack),
			}
			trace = append(trace, step)
			pending = &trace[len(trace)-1]
			return true
		}
		if pending != nil {
			pending.GasAfter = raw.Gas
			if !raw.Halt {
				pending.StackAfter = canonicalStack(raw.Stack)
			}
		}
		return true
	})
	if runErr != nil {
		return ExecutionResult{}, runErr
	}

	status := StatusSuccess
	if intr.Err() != nil {
		status = StatusFault
	} else if intr.IsReverted() {
		status = StatusRevert
	}
	if len(trace) > 0 {
		trace[len(trace)-1].HaltClass = status
		trace[len(trace)-1].StackAfter = nil
	}
	storage := make(map[string]string)
	for _, key := range storageKeys(req, trace) {
		storage[key.Hex()] = state.GetState(contractAddress, key).Hex()
	}
	result := ExecutionResult{
		Engine: "EchoEVM", EngineVersion: moduleVersion("github.com/smallyunet/echoevm"), Status: status,
		ReturnData: "0x" + hex.EncodeToString(intr.ReturnedCode()),
		GasUsed:    req.GasLimit - intr.Gas(), Storage: storage, Trace: trace,
	}
	if intr.Err() != nil {
		result.Error = intr.Err().Error()
	}
	return result, nil
}
