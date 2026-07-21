package differential

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type GethRunner struct{}

func (GethRunner) Run(ctx context.Context, req Request) (ExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return ExecutionResult{}, err
	}
	code, _ := decodeHexField("bytecode", req.Bytecode)
	input, _ := decodeHexField("calldata", req.Calldata)
	state, err := gethstate.New(types.EmptyRootHash, gethstate.NewDatabaseForTesting())
	if err != nil {
		return ExecutionResult{}, err
	}
	state.CreateAccount(contractAddress)
	for key, value := range req.InitialStorage {
		state.SetState(contractAddress, common.HexToHash(key), common.HexToHash(value))
	}
	state.SetCode(contractAddress, code)

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

	cfg := gethRuntimeConfig(req.GasLimit, state, hooks)
	rules := cfg.ChainConfig.Rules(cfg.BlockNumber, cfg.Random != nil, cfg.Time)
	state.Prepare(rules, cfg.Origin, cfg.Coinbase, &contractAddress, gethvm.ActivePrecompiles(rules), nil)
	env := runtime.NewEnv(cfg)
	ret, left, callErr := env.Call(cfg.Origin, contractAddress, input, req.GasLimit, uint256.NewInt(0))
	if output == nil {
		output = ret
	}
	if gasUsed == 0 && left != req.GasLimit {
		gasUsed = req.GasLimit - left
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
		storage[key.Hex()] = state.GetState(contractAddress, key).Hex()
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

func gethRuntimeConfig(gas uint64, state *gethstate.StateDB, hooks *tracing.Hooks) *runtime.Config {
	zero := uint64(0)
	random := common.Hash{}
	chain := &params.ChainConfig{
		ChainID: big.NewInt(1), HomesteadBlock: new(big.Int), EIP150Block: new(big.Int),
		EIP155Block: new(big.Int), EIP158Block: new(big.Int), ByzantiumBlock: new(big.Int),
		ConstantinopleBlock: new(big.Int), PetersburgBlock: new(big.Int), IstanbulBlock: new(big.Int),
		MuirGlacierBlock: new(big.Int), BerlinBlock: new(big.Int), LondonBlock: new(big.Int),
		TerminalTotalDifficulty: new(big.Int), ShanghaiTime: &zero, CancunTime: &zero,
	}
	return &runtime.Config{
		ChainConfig: chain, Difficulty: new(big.Int), BlockNumber: new(big.Int),
		GasLimit: gas, GasPrice: new(big.Int), Value: new(big.Int), BaseFee: new(big.Int),
		BlobBaseFee: big.NewInt(params.BlobTxMinBlobGasprice), Random: &random,
		State: state, EVMConfig: gethvm.Config{Tracer: hooks},
		GetHashFn: func(uint64) common.Hash { return common.Hash{} },
	}
}
