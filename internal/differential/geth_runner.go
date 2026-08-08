package differential

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

type GethRunner struct{}

func (GethRunner) Run(ctx context.Context, req Request) (ExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return ExecutionResult{}, err
	}
	code, _ := decodeHexField("bytecode", req.Bytecode)
	initcode, _ := decodeHexField("initcode", req.InitCode)
	input, _ := decodeHexField("calldata", req.Calldata)
	state, err := gethstate.New(types.EmptyRootHash, gethstate.NewDatabaseForTesting())
	if err != nil {
		return ExecutionResult{}, err
	}
	executionAddress := contractAddress
	if len(initcode) > 0 {
		constructorConfig := gethRuntimeConfig(req.DeployGasLimit, state, nil, req.Fork)
		deployedCode, createdAddress, _, createErr := runtime.Create(initcode, constructorConfig)
		if createErr != nil {
			return ExecutionResult{}, fmt.Errorf("constructor execution failed: %w", createErr)
		}
		if len(deployedCode) == 0 {
			return ExecutionResult{}, errors.New("constructor returned empty runtime bytecode")
		}
		executionAddress = createdAddress
	} else {
		state.CreateAccount(executionAddress)
		state.SetCode(executionAddress, code, tracing.CodeChangeUnspecified)
	}
	for key, value := range req.InitialStorage {
		state.SetState(executionAddress, common.HexToHash(key), common.HexToHash(value))
	}

	trace := make([]NormalizedStep, 0, 128)
	topDepth := -1
	var gasUsed uint64
	var output []byte
	var exitErr error
	traceOverflow := false
	hooks := &tracing.Hooks{}
	hooks.OnOpcode = func(pc uint64, op byte, gas, _ uint64, scope tracing.OpContext, _ []byte, depth int, _ error) {
		if topDepth == -1 {
			topDepth = depth
		}
		if depth != topDepth {
			return
		}
		if len(trace) >= MaxTraceSteps {
			traceOverflow = true
			return
		}
		stack := gethStack(scope)
		if len(trace) > 0 {
			previous := &trace[len(trace)-1]
			previous.GasAfter = gas
			previous.StackAfter = stack
		}
		trace = append(trace, NormalizedStep{
			Index: len(trace), Depth: 0, PC: pc,
			Opcode: "0x" + hex.EncodeToString([]byte{op}), OpcodeName: gethvm.OpCode(op).String(),
			GasBefore: gas, StackBefore: stack,
		})
	}
	hooks.OnExit = func(depth int, out []byte, used uint64, err error, _ bool) {
		if depth != 0 {
			return
		}
		gasUsed, output, exitErr = used, append([]byte(nil), out...), err
	}

	cfg := gethRuntimeConfig(req.GasLimit, state, hooks, req.Fork)
	rules := cfg.ChainConfig.Rules(cfg.BlockNumber, cfg.Random != nil, cfg.Time)
	state.Prepare(rules, cfg.Origin, cfg.Coinbase, &executionAddress, gethvm.ActivePrecompiles(rules), nil)
	env := runtime.NewEnv(cfg)
	initialGas := gethvm.NewGasBudget(req.GasLimit, 0)
	ret, left, callErr := env.Call(cfg.Origin, executionAddress, input, initialGas, uint256.NewInt(0))
	if output == nil {
		output = ret
	}
	if gasUsed == 0 && left.RegularGas != req.GasLimit {
		gasUsed = left.Used(initialGas)
	}
	if exitErr == nil {
		exitErr = callErr
	}
	if traceOverflow {
		return ExecutionResult{}, errors.New("trace exceeds maximum 2000 steps")
	}
	if err := ctx.Err(); err != nil {
		return ExecutionResult{}, err
	}
	if len(trace) > 0 {
		trace[len(trace)-1].GasAfter = req.GasLimit - gasUsed
		trace[len(trace)-1].StackAfter = nil
	}
	status := StatusSuccess
	if errors.Is(exitErr, gethvm.ErrExecutionReverted) || errors.Is(callErr, gethvm.ErrExecutionReverted) {
		status = StatusRevert
	} else if exitErr != nil || callErr != nil {
		status = StatusFault
	}
	if len(trace) > 0 {
		trace[len(trace)-1].HaltClass = status
	}
	storage := make(map[string]string)
	for _, key := range storageKeys(req, trace) {
		storage[key.Hex()] = state.GetState(executionAddress, key).Hex()
	}
	result := ExecutionResult{
		Engine: "Geth", EngineVersion: moduleVersion("github.com/ethereum/go-ethereum"), Status: status,
		ReturnData: "0x" + hex.EncodeToString(output), GasUsed: gasUsed,
		Storage: storage, Trace: trace,
	}
	if exitErr != nil {
		result.Error = exitErr.Error()
	}
	return result, nil
}

func gethStack(scope tracing.OpContext) []string {
	data := scope.StackData()
	out := make([]string, len(data))
	for i := range data {
		out[i] = canonicalWord(data[i].Hex())
	}
	return out
}

func gethRuntimeConfig(gas uint64, state *gethstate.StateDB, hooks *tracing.Hooks, fork string) *runtime.Config {
	zero := uint64(0)
	random := common.Hash{}
	chain := &params.ChainConfig{ChainID: big.NewInt(1)}
	atLeast := func(target string) bool {
		forkIndex, targetIndex := -1, -1
		for index, candidate := range core.SupportedForks {
			if candidate == fork {
				forkIndex = index
			}
			if candidate == target {
				targetIndex = index
			}
		}
		return forkIndex >= targetIndex
	}
	setBlock := func(target string, field **big.Int) {
		if atLeast(target) {
			*field = new(big.Int)
		}
	}
	setTime := func(target string, field **uint64) {
		if atLeast(target) {
			*field = &zero
		}
	}
	setBlock(core.ForkHomestead, &chain.HomesteadBlock)
	setBlock(core.ForkTangerine, &chain.EIP150Block)
	setBlock(core.ForkSpuriousDragon, &chain.EIP155Block)
	setBlock(core.ForkSpuriousDragon, &chain.EIP158Block)
	setBlock(core.ForkByzantium, &chain.ByzantiumBlock)
	setBlock(core.ForkConstantinople, &chain.ConstantinopleBlock)
	setBlock(core.ForkPetersburg, &chain.PetersburgBlock)
	setBlock(core.ForkIstanbul, &chain.IstanbulBlock)
	setBlock(core.ForkBerlin, &chain.BerlinBlock)
	setBlock(core.ForkLondon, &chain.LondonBlock)
	setTime(core.ForkShanghai, &chain.ShanghaiTime)
	setTime(core.ForkCancun, &chain.CancunTime)
	setTime(core.ForkPrague, &chain.PragueTime)
	setTime(core.ForkOsaka, &chain.OsakaTime)
	config := &runtime.Config{
		ChainConfig: chain, Difficulty: new(big.Int), BlockNumber: new(big.Int),
		GasLimit: gas, GasPrice: new(big.Int), Value: new(big.Int), BaseFee: new(big.Int),
		BlobBaseFee: big.NewInt(params.BlobTxMinBlobGasprice),
		State:       state, EVMConfig: gethvm.Config{Tracer: hooks},
		GetHashFn: func(uint64) common.Hash { return common.Hash{} },
	}
	if atLeast(core.ForkParis) {
		config.Random = &random
	}
	return config
}
