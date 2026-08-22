package differential

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	explaintrace "github.com/smallyunet/echoevm/internal/trace"
)

var contractAddress = common.BytesToAddress([]byte("contract"))

type EchoRunner struct{}

func (EchoRunner) Run(ctx context.Context, req Request) (ExecutionResult, error) {
	return (EchoRunner{}).run(ctx, req, nil)
}

// RunExplain executes the same normalized request while collecting structured
// pre/post events across the top-level execution and every nested frame.
func (EchoRunner) RunExplain(ctx context.Context, req Request, maxMemoryBytes int) (ExecutionResult, []explaintrace.OpcodeEvent, error) {
	collector := explaintrace.NewCollector(maxMemoryBytes)
	result, err := (EchoRunner{}).run(ctx, req, collector)
	if err != nil {
		return ExecutionResult{}, nil, err
	}
	return result, collector.Events(), nil
}

func (EchoRunner) run(ctx context.Context, req Request, collector *explaintrace.Collector) (ExecutionResult, error) {
	code, _ := decodeHexField("bytecode", req.Bytecode)
	initcode, _ := decodeHexField("initcode", req.InitCode)
	input, _ := decodeHexField("calldata", req.Calldata)
	state := core.NewMemoryStateDB()
	chainConfig, err := core.ChainConfigForFork(req.Fork)
	if err != nil {
		return ExecutionResult{}, err
	}
	executionAddress := contractAddress
	if len(initcode) > 0 {
		executionAddress = crypto.CreateAddress(common.Address{}, 0)
		state.CreateAccount(executionAddress)
		state.SetNonce(executionAddress, 1)
		state.PrepareTransaction()
		state.AddAddressToAccessList(common.Address{})
		state.AddAddressToAccessList(executionAddress)
		for _, address := range vm.ActivePrecompilesForRules(chainConfig.Rules(new(big.Int))) {
			state.AddAddressToAccessList(address)
		}
		constructor := vm.New(initcode, state, executionAddress)
		configureEchoFork(constructor, chainConfig)
		constructor.SetGas(req.DeployGasLimit)
		constructor.SetBlockGasLimit(req.DeployGasLimit)
		constructor.Run()
		if constructor.Err() != nil {
			return ExecutionResult{}, fmt.Errorf("constructor execution failed: %w", constructor.Err())
		}
		if constructor.IsReverted() {
			return ExecutionResult{}, fmt.Errorf("constructor execution reverted")
		}
		code = constructor.ReturnedCode()
		if len(code) == 0 {
			return ExecutionResult{}, fmt.Errorf("constructor returned empty runtime bytecode")
		}
	}
	state.SetCode(executionAddress, code)
	for key, value := range req.InitialStorage {
		state.InitState(executionAddress, common.HexToHash(key), common.HexToHash(value))
	}
	state.PrepareTransaction()
	state.AddAddressToAccessList(executionAddress)
	for _, address := range vm.ActivePrecompilesForRules(chainConfig.Rules(new(big.Int))) {
		state.AddAddressToAccessList(address)
	}
	executionSnapshot := state.Snapshot()
	intr := vm.NewWithCallData(code, input, state, executionAddress)
	configureEchoFork(intr, chainConfig)
	intr.SetGas(req.GasLimit)
	intr.SetBlockGasLimit(req.GasLimit)
	if collector != nil {
		intr.SetTraceDetails(true)
	}

	trace := make([]NormalizedStep, 0, 128)
	var pending *NormalizedStep
	var runErr error
	explainSteps := 0
	intr.RunWithHook(func(raw vm.TraceStep) bool {
		if err := ctx.Err(); err != nil {
			runErr = err
			return false
		}
		if collector != nil {
			if !raw.IsPost {
				explainSteps++
				if explainSteps > MaxTraceSteps {
					runErr = fmt.Errorf("explainable trace exceeds maximum %d steps", MaxTraceSteps)
					return false
				}
			}
			collector.Consume(raw)
		}
		// The isolated differential contract intentionally compares top-level
		// trace semantics. Nested behavior is still reflected in the parent
		// call/create result, gas, return data, and committed state.
		if raw.Depth != 0 {
			return true
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
	if status != StatusSuccess {
		state.RevertToSnapshot(executionSnapshot)
	}
	storage := make(map[string]string)
	for _, key := range storageKeys(req, trace) {
		storage[key.Hex()] = state.GetState(executionAddress, key).Hex()
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

func configureEchoFork(intr *vm.Interpreter, config *core.ChainConfig) {
	intr.SetChainConfig(config)
	rules := config.Rules(new(big.Int))
	if rules.IsParis {
		intr.SetRandom(new(big.Int))
	}
}
