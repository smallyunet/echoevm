package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/smallyunet/echoevm/internal/rpc"
	"github.com/smallyunet/echoevm/utils"
)

// Package-level logger
var logger zerolog.Logger

func main() {
	cmd, cfg, err := parseFlags()
	if err != nil {
		logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
		logger.Fatal().Err(err).Msg("Failed to parse flags")
		os.Exit(1)
	}

	// Debug information to help troubleshoot
	fmt.Printf("Command: %s\n", cmd)
	fmt.Printf("Config: %+v\n", *cfg)

	lvl, err := zerolog.ParseLevel(strings.ToLower(cfg.LogLevel))
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	cw := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.Kitchen}
	logger = zerolog.New(cw).With().Timestamp().Logger()
	vm.SetLogger(logger)

	switch cmd {
	case "serve":
		fmt.Println("Starting RPC server...")
		runRPCServer(cfg, logger)
		return
	case "range":
		runBlockRange(cfg, logger)
		return
	case "block":
		ctx := context.Background()
		client, err := ethclient.DialContext(ctx, cfg.RPC)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to connect to RPC endpoint")
			os.Exit(1)
		}
		runBlock(ctx, client, cfg.Block, logger)
		return
	}

	// --- Step 1: Load hex encoded bytecode ---
	var hexCode string
	if cfg.Artifact != "" {
		data, err := os.ReadFile(cfg.Artifact)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to read artifact file")
			os.Exit(1)
		}
		var art struct {
			Bytecode         string `json:"bytecode"`
			DeployedBytecode string `json:"deployedBytecode"`
		}
		err = json.Unmarshal(data, &art)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to decode artifact JSON")
			os.Exit(1)
		}
		hexCode = strings.TrimPrefix(art.Bytecode, "0x")
		logger.Info().Msgf("Executing artifact file: %s", cfg.Artifact)
	} else {
		data, err := os.ReadFile(cfg.Bin)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to read bytecode file")
			os.Exit(1)
		}
		hexCode = strings.TrimSpace(string(data))
		logger.Info().Msgf("Executing contract file: %s", cfg.Bin)
	}

	// --- Step 2: Decode hex string to bytecode []byte ---
	code, err := hex.DecodeString(hexCode)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to decode hex bytecode")
		os.Exit(1)
	}

	// --- Step 3: Optional debug output ---
	logger.Debug().Msg("=== Disassembled Bytecode ===")
	utils.PrintBytecode(logger, code, zerolog.DebugLevel)

	// --- Step 4: Create and run the interpreter with constructor bytecode ---
	interpreter := vm.New(code)
	interpreter.Run()

	// --- Step 5: Inspect stack state after constructor execution ---
	switch interpreter.Stack().Len() {
	case 1:
		logger.Info().Msgf("Final Result on Stack: %s", interpreter.Stack().PeekSafe(0).String())
	case 0:
		logger.Info().Msg("Execution finished. Stack is empty.")
	default:
		logger.Info().Msgf("Execution finished. Stack height = %d", interpreter.Stack().Len())
	}

	// --- Step 6: If constructor returned runtime code and mode is "full", execute it ---
	runtimeCode := interpreter.ReturnedCode()
	if cfg.Mode == "full" && len(runtimeCode) > 0 {
		logger.Debug().Msg("=== Runtime Bytecode ===")
		utils.PrintBytecode(logger, runtimeCode, zerolog.DebugLevel)

		var callData []byte
		var err error
		switch {
		case cfg.Calldata != "":
			callData, err = hex.DecodeString(strings.TrimPrefix(cfg.Calldata, "0x"))
		case cfg.Function != "":
			callData, err = buildCallData(cfg.Function, cfg.Args)
		default:
			logger.Fatal().Msg("provide -calldata or -function and -args")
			os.Exit(1)
		}
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to process calldata")
			os.Exit(1)
		}

		runtimeInterpreter := vm.NewWithCallData(runtimeCode, callData)
		runtimeInterpreter.Run()

		switch runtimeInterpreter.Stack().Len() {
		case 1:
			logger.Info().Msgf("Contract %s result: %s", cfg.Bin, runtimeInterpreter.Stack().PeekSafe(0).String())
		case 0:
			logger.Info().Msgf("Contract %s finished. Stack empty.", cfg.Bin)
		default:
			logger.Info().Msgf("Contract %s finished. Stack height = %d", cfg.Bin, runtimeInterpreter.Stack().Len())
		}

		// Exit with error code if execution reverted
		if runtimeInterpreter.IsReverted() {
			os.Exit(1)
		}
	}
}

// check is a helper to panic with context on error - DEPRECATED: Use proper error handling instead
func check(err error, msg string) {
	if err != nil {
		panic(fmt.Sprintf("%s: %v", msg, err))
	}
}

// buildCallData creates ABI encoded calldata from a function signature and
// comma-separated arguments. Only a few basic types (uint256,int256,bool,string)
// are supported. Numeric values can be provided in decimal or 0x-prefixed hex.
func buildCallData(sig, argString string) ([]byte, error) {
	open := strings.Index(sig, "(")
	close := strings.LastIndex(sig, ")")
	if open == -1 || close == -1 || close < open {
		return nil, fmt.Errorf("invalid function signature")
	}
	typesPart := sig[open+1 : close]
	typeNames := []string{}
	if len(typesPart) > 0 {
		for _, t := range strings.Split(typesPart, ",") {
			typeNames = append(typeNames, strings.TrimSpace(t))
		}
	}

	args := []string{}
	if len(argString) > 0 {
		for _, a := range strings.Split(argString, ",") {
			args = append(args, strings.TrimSpace(a))
		}
	}

	if len(typeNames) != len(args) {
		return nil, fmt.Errorf("argument count mismatch")
	}

	var abiArgs abi.Arguments
	values := make([]interface{}, len(args))
	for i, tname := range typeNames {
		t, err := abi.NewType(tname, "", nil)
		if err != nil {
			return nil, err
		}
		abiArgs = append(abiArgs, abi.Argument{Type: t})
		val, err := parseArg(args[i], t)
		if err != nil {
			return nil, err
		}
		values[i] = val
	}
	encoded, err := abiArgs.Pack(values...)
	if err != nil {
		return nil, err
	}
	selector := crypto.Keccak256([]byte(sig))[:4]
	return append(selector, encoded...), nil
}

// parseArg converts a single argument string to the Go value required for ABI
// encoding based on the provided type.
func parseArg(val string, typ abi.Type) (interface{}, error) {
	switch typ.T {
	case abi.UintTy, abi.IntTy:
		n := new(big.Int)
		var ok bool
		if strings.HasPrefix(val, "0x") {
			n, ok = n.SetString(val[2:], 16)
		} else {
			n, ok = n.SetString(val, 10)
		}
		if !ok {
			return nil, fmt.Errorf("invalid integer value: %s", val)
		}
		return n, nil
	case abi.BoolTy:
		return strings.ToLower(val) == "true", nil
	case abi.StringTy:
		return val, nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", typ.String())
	}
}

// runBlock connects to an Ethereum RPC endpoint and executes all contract
// transactions in the specified block using the echoevm interpreter.
func runBlock(ctx context.Context, client *ethclient.Client, blockNum int, logger zerolog.Logger) {
	bnum := big.NewInt(int64(blockNum))
	block, err := client.BlockByNumber(ctx, bnum)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to fetch block")
		os.Exit(1)
	}

	contractTxs := []*types.Transaction{}
	for _, tx := range block.Transactions() {
		data := tx.Data()
		if len(data) == 0 {
			continue
		}

		if tx.To() == nil {
			contractTxs = append(contractTxs, tx)
			continue
		}

		code, err := client.CodeAt(ctx, *tx.To(), bnum)
		if err == nil && len(code) > 0 {
			contractTxs = append(contractTxs, tx)
		}
	}

	logger.Info().Msgf("Block %d contains %d contract transactions", blockNum, len(contractTxs))

	run := func(i *vm.Interpreter) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%v", r)
			}
		}()
		i.Run()
		return nil
	}

	success := 0
	for idx, tx := range contractTxs {
		data := tx.Data()
		if tx.To() == nil {
			logger.Info().Msgf("tx %d: contract creation", idx)
			interpreter := vm.New(data)
			interpreter.SetBlockNumber(block.NumberU64())
			if err := run(interpreter); err != nil {
				logger.Error().Msgf("tx %d failed: %v", idx, err)
				continue
			}
			success++
			logger.Info().Msgf("stack height %d", interpreter.Stack().Len())
			continue
		}

		code, err := client.CodeAt(ctx, *tx.To(), bnum)
		if err != nil || len(code) == 0 {
			logger.Warn().Msgf("tx %d: missing contract code", idx)
			continue
		}
		logger.Info().Msgf("tx %d: call %s", idx, tx.To().Hex())
		interpreter := vm.NewWithCallData(code, data)
		interpreter.SetBlockNumber(block.NumberU64())
		if err := run(interpreter); err != nil {
			logger.Error().Msgf("tx %d failed: %v", idx, err)
			continue
		}
		success++
		logger.Info().Msgf("stack height %d", interpreter.Stack().Len())
	}

	logger.Info().Msgf("Executed block %d - %d/%d transactions succeeded", blockNum, success, len(contractTxs))
}

func runBlockRange(cfg *cliConfig, logger zerolog.Logger) {
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, cfg.RPC)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to RPC endpoint")
		os.Exit(1)
	}
	for n := cfg.StartBlock; n <= cfg.EndBlock; n++ {
		logger.Info().Msgf("=== Executing block %d ===", n)
		runBlock(ctx, client, n, logger)
	}
}

// runRPCServer starts a JSON-RPC server compatible with Geth
func runRPCServer(cfg *cliConfig, logger zerolog.Logger) {
	logger.Info().Msgf("Starting EchoEVM RPC server on %s", cfg.RPCEndpoint)

	// Create new RPC server instance
	rpcServer := rpc.NewServer(cfg.RPCEndpoint, logger)

	// Start the server
	err := rpcServer.Start()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to start RPC server")
	}

	// Wait for interrupt signal to gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal
	<-c

	// Shutdown the server
	logger.Info().Msg("Shutting down RPC server...")
	err = rpcServer.Stop()
	if err != nil {
		logger.Error().Err(err).Msg("Error shutting down RPC server")
	}
}
