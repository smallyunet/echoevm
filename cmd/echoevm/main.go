package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/smallyunet/echoevm/utils"
)

func main() {
	cfg := parseFlags()
	lvl, err := zerolog.ParseLevel(strings.ToLower(cfg.LogLevel))
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	logger := zerologlog.Logger
	vm.SetLogger(logger)

	// --- Step 1: Read hex-encoded constructor bytecode from file ---
	data, err := os.ReadFile(cfg.Bin)
	check(err, "failed to read bytecode file")

	// --- Step 2: Decode hex string to bytecode []byte ---
	code, err := hex.DecodeString(string(data))
	check(err, "failed to decode hex bytecode")

	// --- Step 3: Optional debug output ---
	logger.Info().Msg("=== Disassembled Bytecode ===")
	utils.PrintBytecode(logger, code)

	// --- Step 4: Create and run the interpreter with constructor bytecode ---
	interpreter := vm.New(code)
	interpreter.Run()

	// --- Step 5: Inspect stack state after constructor execution ---
	switch interpreter.Stack().Len() {
	case 1:
		logger.Info().Msgf("Final Result on Stack: %s", interpreter.Stack().Peek(0).String())
	case 0:
		logger.Info().Msg("Execution finished. Stack is empty.")
	default:
		logger.Info().Msgf("Execution finished. Stack height = %d", interpreter.Stack().Len())
	}

	// --- Step 6: If constructor returned runtime code and mode is "full", execute it ---
	runtimeCode := interpreter.ReturnedCode()
	if cfg.Mode == "full" && len(runtimeCode) > 0 {
		logger.Info().Msg("=== Runtime Bytecode ===")
		utils.PrintBytecode(logger, runtimeCode)

		var callData []byte
		var err error
		switch {
		case cfg.Calldata != "":
			callData, err = hex.DecodeString(strings.TrimPrefix(cfg.Calldata, "0x"))
		case cfg.Function != "" && cfg.Args != "":
			callData, err = buildCallData(cfg.Function, cfg.Args)
		default:
			callData, _ = hex.DecodeString("771602f7000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002")
		}
		check(err, "failed to process calldata")

		runtimeInterpreter := vm.NewWithCallData(runtimeCode, callData)
		runtimeInterpreter.Run()

		switch runtimeInterpreter.Stack().Len() {
		case 1:
			logger.Info().Msgf("Runtime Result on Stack: %s", runtimeInterpreter.Stack().Peek(0).String())
		case 0:
			logger.Info().Msg("Runtime execution finished. Stack is empty.")
		default:
			logger.Info().Msgf("Runtime execution finished. Stack height = %d", runtimeInterpreter.Stack().Len())
		}
	}
}

// check is a helper to panic with context on error
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
