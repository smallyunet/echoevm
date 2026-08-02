package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/smallyunet/echoevm/internal/differential"
	"github.com/spf13/cobra"
)

type solidityRunFlags struct {
	contract        string
	function        string
	args            string
	constructorArgs string
	solc            string
	basePath        string
	includePaths    []string
	gas             uint64
	format          string
	trace           bool
	diff            bool
	optimize        bool
}

type compiledSolidityContract struct {
	key                 string
	name                string
	constructorBytecode string
	runtimeBytecode     string
	abi                 gethabi.ABI
}

type solidityRunOutput struct {
	Source     string                         `json:"source"`
	Contract   string                         `json:"contract"`
	Function   string                         `json:"function"`
	Execution  differential.ExecutionResult   `json:"execution"`
	Comparison *differential.ComparisonResult `json:"comparison,omitempty"`
}

func newSolidityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "solidity",
		Short: "Compile and execute Solidity source in an isolated EVM",
	}
	cmd.AddCommand(newSolidityRunCmd())
	return cmd
}

func newSolidityRunCmd() *cobra.Command {
	flags := &solidityRunFlags{}
	cmd := &cobra.Command{
		Use:     "run <source.sol>",
		Short:   "Compile a Solidity contract and execute one function",
		Args:    cobra.ExactArgs(1),
		Example: "echoevm solidity run Counter.sol --contract Counter --function 'add(uint256,uint256)' --args 2,40 --diff",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSolidity(cmd.Context(), cmd.OutOrStdout(), args[0], flags)
		},
	}
	cmd.Flags().StringVar(&flags.contract, "contract", "", "contract name (required when the source produces multiple deployable contracts)")
	cmd.Flags().StringVarP(&flags.function, "function", "f", "run", "function name or canonical signature")
	cmd.Flags().StringVarP(&flags.args, "args", "A", "", "comma-separated function arguments")
	cmd.Flags().StringVar(&flags.constructorArgs, "constructor-args", "", "comma-separated constructor arguments")
	cmd.Flags().StringVar(&flags.solc, "solc", "solc", "Solidity compiler executable")
	cmd.Flags().StringVar(&flags.basePath, "base-path", "", "solc base path (defaults to the source directory)")
	cmd.Flags().StringSliceVar(&flags.includePaths, "include-path", nil, "additional solc import path (repeatable or comma-separated)")
	cmd.Flags().Uint64Var(&flags.gas, "gas", differential.DefaultGasLimit, "execution gas limit")
	cmd.Flags().StringVar(&flags.format, "format", "text", "output format (text|json)")
	cmd.Flags().BoolVar(&flags.trace, "trace", false, "include the EchoEVM opcode trace")
	cmd.Flags().BoolVar(&flags.diff, "diff", false, "compare EchoEVM execution with embedded Geth")
	cmd.Flags().BoolVar(&flags.optimize, "optimize", false, "enable the Solidity optimizer")
	return cmd
}

func runSolidity(ctx context.Context, out io.Writer, source string, flags *solidityRunFlags) error {
	if flags.format != "text" && flags.format != "json" {
		return fmt.Errorf("unsupported format %q: use text or json", flags.format)
	}
	compiled, err := compileSolidity(ctx, source, flags)
	if err != nil {
		return err
	}
	contract, err := selectCompiledContract(compiled, flags.contract)
	if err != nil {
		return err
	}
	method, err := resolveSolidityMethod(contract.abi, flags.function)
	if err != nil {
		return err
	}
	calldata, err := buildCallData(method.Sig, flags.args)
	if err != nil {
		return fmt.Errorf("encode %s arguments: %w", method.Sig, err)
	}
	initcode, err := buildConstructorData(contract, flags.constructorArgs)
	if err != nil {
		return err
	}

	req := differential.Request{
		Fork: differential.ForkCancun, Bytecode: contract.runtimeBytecode,
		InitCode: initcode, Calldata: fmt.Sprintf("0x%x", calldata), GasLimit: flags.gas,
	}
	engine := differential.DefaultEngine()
	result := solidityRunOutput{
		Source: source, Contract: contract.name, Function: method.Sig,
	}
	if flags.diff {
		comparison, compareErr := engine.Compare(ctx, req)
		if compareErr != nil {
			return compareErr
		}
		result.Execution = comparison.EchoEVM
		result.Comparison = &comparison
	} else {
		execution, runErr := engine.RunEcho(ctx, req)
		if runErr != nil {
			return runErr
		}
		result.Execution = execution
	}

	if err := writeSolidityRunOutput(out, result, flags); err != nil {
		return err
	}
	if result.Execution.Status != differential.StatusSuccess {
		return fmt.Errorf("execution %s", result.Execution.Status)
	}
	return nil
}

func compileSolidity(ctx context.Context, source string, flags *solidityRunFlags) ([]compiledSolidityContract, error) {
	absSource, err := filepath.Abs(source)
	if err != nil {
		return nil, fmt.Errorf("resolve Solidity source: %w", err)
	}
	info, err := os.Stat(absSource)
	if err != nil {
		return nil, fmt.Errorf("read Solidity source: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("Solidity source is a directory: %s", source)
	}

	basePath := flags.basePath
	if basePath == "" {
		basePath = filepath.Dir(absSource)
	}
	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("resolve solc base path: %w", err)
	}
	args := []string{"--combined-json", "abi,bin,bin-runtime", "--evm-version", "cancun", "--base-path", absBasePath}
	for _, includePath := range flags.includePaths {
		absolute, pathErr := filepath.Abs(includePath)
		if pathErr != nil {
			return nil, fmt.Errorf("resolve solc include path %q: %w", includePath, pathErr)
		}
		args = append(args, "--include-path", absolute)
	}
	if flags.optimize {
		args = append(args, "--optimize")
	}
	args = append(args, absSource)

	command := exec.CommandContext(ctx, flags.solc, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("solc compilation canceled: %w", ctx.Err())
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("solc compilation failed: %s", detail)
	}

	var combined struct {
		Contracts map[string]struct {
			ABI        json.RawMessage `json:"abi"`
			Bin        string          `json:"bin"`
			BinRuntime string          `json:"bin-runtime"`
		} `json:"contracts"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &combined); err != nil {
		return nil, fmt.Errorf("parse solc output: %w", err)
	}
	contracts := make([]compiledSolidityContract, 0, len(combined.Contracts))
	for key, artifact := range combined.Contracts {
		if artifact.BinRuntime == "" {
			continue
		}
		parsedABI, err := parseCombinedJSONABI(artifact.ABI)
		if err != nil {
			return nil, fmt.Errorf("parse ABI for %s: %w", key, err)
		}
		name := key
		if colon := strings.LastIndex(key, ":"); colon >= 0 {
			name = key[colon+1:]
		}
		contracts = append(contracts, compiledSolidityContract{
			key: key, name: name, constructorBytecode: "0x" + artifact.Bin,
			runtimeBytecode: "0x" + artifact.BinRuntime, abi: parsedABI,
		})
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].key < contracts[j].key })
	if len(contracts) == 0 {
		return nil, fmt.Errorf("solc produced no deployable contracts for %s", source)
	}
	return contracts, nil
}

func parseCombinedJSONABI(raw json.RawMessage) (gethabi.ABI, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return gethabi.ABI{}, fmt.Errorf("missing ABI")
	}
	if raw[0] == '"' {
		var encoded string
		if err := json.Unmarshal(raw, &encoded); err != nil {
			return gethabi.ABI{}, err
		}
		raw = []byte(encoded)
	}
	return gethabi.JSON(bytes.NewReader(raw))
}

func selectCompiledContract(contracts []compiledSolidityContract, requested string) (compiledSolidityContract, error) {
	if requested != "" {
		matches := make([]compiledSolidityContract, 0, 1)
		for _, contract := range contracts {
			if contract.name == requested || contract.key == requested {
				matches = append(matches, contract)
			}
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		if len(matches) > 1 {
			keys := make([]string, len(matches))
			for i, contract := range matches {
				keys[i] = contract.key
			}
			return compiledSolidityContract{}, fmt.Errorf("contract name %q is ambiguous; use a source-qualified name: %s", requested, strings.Join(keys, ", "))
		}
		return compiledSolidityContract{}, fmt.Errorf("contract %q not found; available: %s", requested, compiledContractNames(contracts))
	}
	if len(contracts) != 1 {
		return compiledSolidityContract{}, fmt.Errorf("source produced multiple deployable contracts; choose one with --contract: %s", compiledContractNames(contracts))
	}
	return contracts[0], nil
}

func compiledContractNames(contracts []compiledSolidityContract) string {
	names := make([]string, len(contracts))
	for i, contract := range contracts {
		names[i] = contract.name
	}
	return strings.Join(names, ", ")
}

func resolveSolidityMethod(contractABI gethabi.ABI, requested string) (gethabi.Method, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = "run"
	}
	var matches []gethabi.Method
	for _, method := range contractABI.Methods {
		if method.Sig == requested || (!strings.Contains(requested, "(") && method.RawName == requested) {
			matches = append(matches, method)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		signatures := make([]string, len(matches))
		for i, method := range matches {
			signatures[i] = method.Sig
		}
		sort.Strings(signatures)
		return gethabi.Method{}, fmt.Errorf("function %q is overloaded; use a canonical signature: %s", requested, strings.Join(signatures, ", "))
	}
	return gethabi.Method{}, fmt.Errorf("function %q not found in contract ABI", requested)
}

func buildConstructorData(contract compiledSolidityContract, argString string) (string, error) {
	if contract.constructorBytecode == "" || contract.constructorBytecode == "0x" {
		return "", fmt.Errorf("contract %s has no deployable constructor bytecode", contract.name)
	}
	arguments := contract.abi.Constructor.Inputs
	values, err := parseABIArguments(arguments, argString)
	if err != nil {
		return "", fmt.Errorf("encode constructor arguments: %w", err)
	}
	encoded, err := arguments.Pack(values...)
	if err != nil {
		return "", fmt.Errorf("encode constructor arguments: %w", err)
	}
	return contract.constructorBytecode + fmt.Sprintf("%x", encoded), nil
}

func parseABIArguments(arguments gethabi.Arguments, argString string) ([]interface{}, error) {
	valuesText := []string{}
	if argString != "" {
		valuesText = splitArgs(argString)
	}
	if len(arguments) != len(valuesText) {
		return nil, fmt.Errorf("argument count mismatch: expected %d, got %d", len(arguments), len(valuesText))
	}
	values := make([]interface{}, len(arguments))
	for i, argument := range arguments {
		value, err := parseArg(valuesText[i], argument.Type)
		if err != nil {
			return nil, fmt.Errorf("argument %d (%s): %w", i, argument.Type.String(), err)
		}
		values[i] = value
	}
	return values, nil
}

func writeSolidityRunOutput(out io.Writer, result solidityRunOutput, flags *solidityRunFlags) error {
	if flags.format == "json" {
		if !flags.trace {
			result.Execution.Trace = nil
			if result.Comparison != nil {
				result.Comparison.EchoEVM.Trace = nil
				result.Comparison.Geth.Trace = nil
			}
		}
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	if _, err := fmt.Fprintf(out, "EchoEVM Solidity run — %s:%s %s\n", result.Source, result.Contract, result.Function); err != nil {
		return err
	}
	if result.Comparison != nil {
		if err := writeDiffText(out, *result.Comparison); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintf(out, "status=%s return=%s gas=%d storage=%d\n", result.Execution.Status, result.Execution.ReturnData, result.Execution.GasUsed, len(result.Execution.Storage)); err != nil {
		return err
	}
	if flags.trace {
		for _, step := range result.Execution.Trace {
			if _, err := fmt.Fprintf(out, "%04d pc=%04d op=%-12s gas=%d stack=%v\n", step.Index, step.PC, step.OpcodeName, step.GasBefore, step.StackBefore); err != nil {
				return err
			}
		}
	}
	return nil
}
