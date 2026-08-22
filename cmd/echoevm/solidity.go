package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/smallyunet/echoevm/internal/differential"
	explaintrace "github.com/smallyunet/echoevm/internal/trace"
	"github.com/spf13/cobra"
	ethabi "github.com/umbracle/ethgo/abi"
)

const solidityProtocolVersion = 1

type solidityRunFlags struct {
	contract        string
	function        string
	args            string
	constructorArgs string
	solc            string
	solcArgs        []string
	basePath        string
	includePaths    []string
	gas             uint64
	deployGas       uint64
	format          string
	trace           bool
	optimize        bool
	optimizerRuns   uint64
	viaIR           bool
	remappings      []string
	profile         string
	limit           int
	maxMemoryBytes  int
}

type compiledSolidityContract struct {
	key                 string
	name                string
	constructorBytecode string
	runtimeBytecode     string
	runtimeSourceMap    string
	abi                 *ethabi.ABI
	mutabilities        map[string]string
	functionLocations   map[string]soliditySourceLocation
	sourceNames         map[int]string
}

type soliditySourceLocation struct {
	File   string `json:"file"`
	Start  int    `json:"start"`
	Length int    `json:"length"`
}

type solidityPCSourceLocation struct {
	PC     uint64 `json:"pc"`
	File   string `json:"file"`
	Start  int    `json:"start"`
	Length int    `json:"length"`
}

type solidityRuntimeSourceMap struct {
	Locations []solidityPCSourceLocation `json:"locations"`
}

type solidityRunOutput struct {
	SchemaVersion int                            `json:"schemaVersion"`
	Source        string                         `json:"source"`
	Contract      string                         `json:"contract"`
	Function      string                         `json:"function"`
	Compiler      solidityCompilerInfo           `json:"compiler"`
	DurationMS    int64                          `json:"durationMs"`
	Execution     differential.ExecutionResult   `json:"execution"`
	Evidence      *explaintrace.EvidenceDocument `json:"-"`
	SourceMap     *solidityRuntimeSourceMap      `json:"-"`
}

type solidityEvidenceJSONOutput struct {
	explaintrace.EvidenceDocument
	Source   string               `json:"source"`
	Contract string               `json:"contract"`
	Function string               `json:"function"`
	Compiler solidityCompilerInfo `json:"compiler"`
}

type solidityCompilerInfo struct {
	Executable string `json:"executable"`
	Version    string `json:"version"`
}

type solidityRunJSONOutput struct {
	SchemaVersion int                          `json:"schemaVersion"`
	Source        string                       `json:"source"`
	Contract      string                       `json:"contract"`
	Function      string                       `json:"function"`
	Compiler      solidityCompilerInfo         `json:"compiler"`
	DurationMS    int64                        `json:"durationMs"`
	Execution     differential.ExecutionResult `json:"execution"`
	SourceMap     *solidityRuntimeSourceMap    `json:"sourceMap,omitempty"`
}

type solidityRunSummaryJSONOutput struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Source        string               `json:"source"`
	Contract      string               `json:"contract"`
	Function      string               `json:"function"`
	Compiler      solidityCompilerInfo `json:"compiler"`
	DurationMS    int64                `json:"durationMs"`
	Execution     *executionSummary    `json:"execution,omitempty"`
}

type solidityParameterOutput struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type"`
}

type solidityFunctionOutput struct {
	Name            string                    `json:"name"`
	Signature       string                    `json:"signature"`
	Inputs          []solidityParameterOutput `json:"inputs"`
	Outputs         []solidityParameterOutput `json:"outputs"`
	StateMutability string                    `json:"stateMutability"`
	SourceLocation  *soliditySourceLocation   `json:"sourceLocation,omitempty"`
}

type solidityContractOutput struct {
	Key         string                    `json:"key"`
	Name        string                    `json:"name"`
	Constructor []solidityParameterOutput `json:"constructorInputs"`
	Functions   []solidityFunctionOutput  `json:"functions"`
}

type solidityInspectOutput struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Source        string                   `json:"source"`
	Compiler      solidityCompilerInfo     `json:"compiler"`
	DurationMS    int64                    `json:"durationMs"`
	Contracts     []solidityContractOutput `json:"contracts"`
}

type solidityErrorOutput struct {
	SchemaVersion int `json:"schemaVersion"`
	Error         struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type solcASTNode struct {
	NodeType         string        `json:"nodeType"`
	Name             string        `json:"name"`
	Src              string        `json:"src"`
	FunctionSelector string        `json:"functionSelector"`
	Nodes            []solcASTNode `json:"nodes"`
}

type reportedSolidityError struct{ message string }

func (e reportedSolidityError) Error() string { return e.message }

func newSolidityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "solidity",
		Short: "Compile and execute Solidity source in an isolated EVM",
	}
	cmd.AddCommand(newSolidityRunCmd())
	cmd.AddCommand(newSolidityInspectCmd())
	return cmd
}

func newSolidityRunCmd() *cobra.Command {
	flags := &solidityRunFlags{}
	cmd := &cobra.Command{
		Use:     "run <source.sol>",
		Short:   "Compile a Solidity contract and execute one function",
		Args:    cobra.ExactArgs(1),
		Example: "echoevm solidity run Counter.sol --contract Counter --function 'add(uint256,uint256)' --args 2,40",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runSolidity(cmd.Context(), cmd.OutOrStdout(), args[0], flags)
			if err != nil && (flags.format == "json" || flags.format == "summary-json" || flags.format == "evidence-json") {
				if _, alreadyReported := err.(reportedSolidityError); !alreadyReported {
					_ = writeSolidityError(cmd.OutOrStdout(), classifySolidityError(err), err)
				}
			}
			return err
		},
	}
	cmd.Flags().StringVar(&flags.contract, "contract", "", "contract name (required when the source produces multiple deployable contracts)")
	cmd.Flags().StringVarP(&flags.function, "function", "f", "run", "function name or canonical signature")
	cmd.Flags().StringVarP(&flags.args, "args", "A", "", "comma-separated function arguments")
	cmd.Flags().StringVar(&flags.constructorArgs, "constructor-args", "", "comma-separated constructor arguments")
	cmd.Flags().StringVar(&flags.solc, "solc", "solc", "Solidity compiler executable")
	cmd.Flags().StringArrayVar(&flags.solcArgs, "solc-arg", nil, "argument placed before solc compilation options (repeatable)")
	cmd.Flags().StringVar(&flags.basePath, "base-path", "", "solc base path (defaults to the source directory)")
	cmd.Flags().StringSliceVar(&flags.includePaths, "include-path", nil, "additional solc import path (repeatable or comma-separated)")
	cmd.Flags().Uint64Var(&flags.gas, "gas", differential.DefaultGasLimit, "execution gas limit")
	cmd.Flags().Uint64Var(&flags.deployGas, "deploy-gas", 0, "constructor deployment gas limit (defaults to --gas)")
	cmd.Flags().StringVar(&flags.format, "format", "text", "output format (text|json|summary-json|evidence-json)")
	cmd.Flags().BoolVar(&flags.trace, "trace", false, "include the EchoEVM opcode trace")
	cmd.Flags().BoolVar(&flags.optimize, "optimize", false, "enable the Solidity optimizer")
	cmd.Flags().Uint64Var(&flags.optimizerRuns, "optimizer-runs", 0, "Solidity optimizer runs (Foundry auto-detected when omitted)")
	cmd.Flags().BoolVar(&flags.viaIR, "via-ir", false, "compile through the Solidity IR pipeline")
	cmd.Flags().StringArrayVar(&flags.remappings, "remapping", nil, "Solidity import remapping (repeatable; Foundry auto-detected)")
	cmd.Flags().StringVar(&flags.profile, "profile", explaintrace.ProfileAuto, "Evidence profile for evidence-json (auto|revert|storage|call|abi|gas|arithmetic|full)")
	cmd.Flags().IntVar(&flags.limit, "limit", 40, "Maximum evidence events; execution still completes (0 = no limit)")
	cmd.Flags().IntVar(&flags.maxMemoryBytes, "max-memory-bytes", 256, "Maximum changed memory bytes captured per opcode")
	return cmd
}

func newSolidityInspectCmd() *cobra.Command {
	flags := &solidityRunFlags{}
	cmd := &cobra.Command{
		Use:   "inspect <source.sol>",
		Short: "List deployable contracts and ABI functions as editor-friendly JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runSolidityInspect(cmd.Context(), cmd.OutOrStdout(), args[0], flags)
			if err != nil && flags.format == "json" {
				_ = writeSolidityError(cmd.OutOrStdout(), classifySolidityError(err), err)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&flags.solc, "solc", "solc", "Solidity compiler executable")
	cmd.Flags().StringArrayVar(&flags.solcArgs, "solc-arg", nil, "argument placed before solc compilation options (repeatable)")
	cmd.Flags().StringVar(&flags.basePath, "base-path", "", "solc base path (defaults to the source directory)")
	cmd.Flags().StringSliceVar(&flags.includePaths, "include-path", nil, "additional solc import path (repeatable or comma-separated)")
	cmd.Flags().StringVar(&flags.format, "format", "json", "output format (text|json)")
	cmd.Flags().BoolVar(&flags.optimize, "optimize", false, "enable the Solidity optimizer")
	cmd.Flags().Uint64Var(&flags.optimizerRuns, "optimizer-runs", 0, "Solidity optimizer runs (Foundry auto-detected when omitted)")
	cmd.Flags().BoolVar(&flags.viaIR, "via-ir", false, "compile through the Solidity IR pipeline")
	cmd.Flags().StringArrayVar(&flags.remappings, "remapping", nil, "Solidity import remapping (repeatable; Foundry auto-detected)")
	return cmd
}

func runSolidity(ctx context.Context, out io.Writer, source string, flags *solidityRunFlags) error {
	startedAt := time.Now()
	if flags.format != "text" && flags.format != "json" && flags.format != "summary-json" && flags.format != "evidence-json" {
		return fmt.Errorf("unsupported format %q: use text, json, summary-json, or evidence-json", flags.format)
	}
	if flags.format == "evidence-json" {
		if flags.limit < 0 || flags.maxMemoryBytes < 0 {
			return fmt.Errorf("evidence limits must be non-negative")
		}
		if err := explaintrace.ValidateEvidenceProfile(flags.profile); err != nil {
			return err
		}
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
	calldata, err := buildCallData(method.Sig(), flags.args)
	if err != nil {
		return fmt.Errorf("encode %s arguments: %w", method.Sig(), err)
	}
	initcode, err := buildConstructorData(contract, flags.constructorArgs)
	if err != nil {
		return err
	}

	req := differential.Request{
		Fork: differential.ForkOsaka, Bytecode: contract.runtimeBytecode,
		InitCode: initcode, Calldata: fmt.Sprintf("0x%x", calldata), GasLimit: flags.gas,
		DeployGasLimit: flags.deployGas,
	}
	engine := differential.DefaultEngine()
	result := solidityRunOutput{
		SchemaVersion: solidityProtocolVersion,
		Source:        source, Contract: contract.name, Function: method.Sig(),
		Compiler:  solidityCompilerInfo{Executable: flags.solc, Version: solidityCompilerVersion(ctx, flags.solc, flags.solcArgs)},
		SourceMap: buildRuntimeSourceMap(contract),
	}
	if flags.format == "evidence-json" {
		execution, events, explainErr := engine.RunEchoExplain(ctx, req, flags.maxMemoryBytes)
		if explainErr != nil {
			return explainErr
		}
		result.Execution = execution
		document, evidenceErr := explaintrace.BuildEvidence(explaintrace.ExecutionResult{
			Status: string(execution.Status), GasLimit: flags.gas, GasUsed: execution.GasUsed,
			ReturnData: execution.ReturnData, TotalSteps: len(events), MatchedSteps: len(events), EmittedSteps: len(events),
			ExecutionError: execution.Error,
		}, events, flags.profile, flags.limit)
		if evidenceErr != nil {
			return evidenceErr
		}
		result.Evidence = &document
	} else {
		execution, runErr := engine.RunEcho(ctx, req)
		if runErr != nil {
			return runErr
		}
		result.Execution = execution
	}

	result.DurationMS = time.Since(startedAt).Milliseconds()
	if err := writeSolidityRunOutput(out, result, flags); err != nil {
		return err
	}
	if result.Execution.Status != differential.StatusSuccess {
		return reportedSolidityError{message: fmt.Sprintf("execution %s", result.Execution.Status)}
	}
	return nil
}

func runSolidityInspect(ctx context.Context, out io.Writer, source string, flags *solidityRunFlags) error {
	startedAt := time.Now()
	if flags.format != "text" && flags.format != "json" {
		return fmt.Errorf("unsupported format %q: use text or json", flags.format)
	}
	compiled, err := compileSolidity(ctx, source, flags)
	if err != nil {
		return err
	}
	result := solidityInspectOutput{
		SchemaVersion: solidityProtocolVersion,
		Source:        source,
		Compiler:      solidityCompilerInfo{Executable: flags.solc, Version: solidityCompilerVersion(ctx, flags.solc, flags.solcArgs)},
		Contracts:     make([]solidityContractOutput, 0, len(compiled)),
	}
	for _, contract := range compiled {
		item := solidityContractOutput{
			Key: contract.key, Name: contract.name,
			Constructor: solidityParameters(constructorInputs(contract.abi)),
		}
		for _, method := range contract.abi.Methods {
			location := contract.functionLocations[fmt.Sprintf("%x", method.ID())]
			var sourceLocation *soliditySourceLocation
			if location.File != "" {
				copy := location
				sourceLocation = &copy
			}
			item.Functions = append(item.Functions, solidityFunctionOutput{
				Name: method.Name, Signature: method.Sig(),
				Inputs: solidityParameters(method.Inputs), Outputs: solidityParameters(method.Outputs),
				StateMutability: solidityMutability(method, contract.mutabilities[method.Sig()]),
				SourceLocation:  sourceLocation,
			})
		}
		sort.Slice(item.Functions, func(i, j int) bool { return item.Functions[i].Signature < item.Functions[j].Signature })
		result.Contracts = append(result.Contracts, item)
	}
	result.DurationMS = time.Since(startedAt).Milliseconds()
	if flags.format == "json" {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	for _, contract := range result.Contracts {
		if _, err := fmt.Fprintf(out, "%s (%s)\n", contract.Name, contract.Key); err != nil {
			return err
		}
		for _, method := range contract.Functions {
			if _, err := fmt.Fprintf(out, "  %s\n", method.Signature); err != nil {
				return err
			}
		}
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
		return nil, fmt.Errorf("solidity source is a directory: %s", source)
	}

	basePath := flags.basePath
	if basePath == "" {
		basePath = filepath.Dir(absSource)
	}
	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("resolve solc base path: %w", err)
	}
	compilerSettings, err := resolveSolidityCompilerSettings(absBasePath, flags)
	if err != nil {
		return nil, err
	}
	sourceKey, err := filepath.Rel(absBasePath, absSource)
	if err != nil || strings.HasPrefix(sourceKey, ".."+string(filepath.Separator)) || sourceKey == ".." {
		sourceKey = absSource
	}
	sourceKey = filepath.ToSlash(sourceKey)
	sourceContents, err := os.ReadFile(absSource)
	if err != nil {
		return nil, fmt.Errorf("read Solidity source: %w", err)
	}
	compilerInput := struct {
		Language string `json:"language"`
		Sources  map[string]struct {
			Content string `json:"content"`
		} `json:"sources"`
		Settings struct {
			Optimizer struct {
				Enabled bool   `json:"enabled"`
				Runs    uint64 `json:"runs,omitempty"`
			} `json:"optimizer"`
			ViaIR           bool                           `json:"viaIR,omitempty"`
			Remappings      []string                       `json:"remappings,omitempty"`
			EVMVersion      string                         `json:"evmVersion"`
			OutputSelection map[string]map[string][]string `json:"outputSelection"`
		} `json:"settings"`
	}{Language: "Solidity", Sources: make(map[string]struct {
		Content string `json:"content"`
	})}
	compilerInput.Sources[sourceKey] = struct {
		Content string `json:"content"`
	}{Content: string(sourceContents)}
	compilerInput.Settings.Optimizer.Enabled = compilerSettings.Optimize
	compilerInput.Settings.Optimizer.Runs = compilerSettings.OptimizerRuns
	compilerInput.Settings.ViaIR = compilerSettings.ViaIR
	compilerInput.Settings.Remappings = compilerSettings.Remappings
	compilerInput.Settings.EVMVersion = "cancun"
	compilerInput.Settings.OutputSelection = map[string]map[string][]string{
		"*": {
			"*": {"abi", "evm.bytecode.object", "evm.deployedBytecode.object", "evm.deployedBytecode.sourceMap"},
			"":  {"ast"},
		},
	}
	standardJSON, err := json.Marshal(compilerInput)
	if err != nil {
		return nil, fmt.Errorf("encode solc input: %w", err)
	}

	args := append([]string{}, flags.solcArgs...)
	args = append(args, "--standard-json", "--base-path", absBasePath)
	for _, includePath := range flags.includePaths {
		absolute, pathErr := filepath.Abs(includePath)
		if pathErr != nil {
			return nil, fmt.Errorf("resolve solc include path %q: %w", includePath, pathErr)
		}
		args = append(args, "--include-path", absolute)
	}

	command := exec.CommandContext(ctx, flags.solc, args...)
	var stdout, stderr bytes.Buffer
	command.Stdin = bytes.NewReader(standardJSON)
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

	var compiledOutput struct {
		Contracts map[string]map[string]struct {
			ABI json.RawMessage `json:"abi"`
			EVM struct {
				Bytecode struct {
					Object string `json:"object"`
				} `json:"bytecode"`
				DeployedBytecode struct {
					Object    string `json:"object"`
					SourceMap string `json:"sourceMap"`
				} `json:"deployedBytecode"`
			} `json:"evm"`
		} `json:"contracts"`
		Sources map[string]struct {
			ID  int         `json:"id"`
			AST solcASTNode `json:"ast"`
		} `json:"sources"`
		Errors []struct {
			Severity         string `json:"severity"`
			Message          string `json:"message"`
			FormattedMessage string `json:"formattedMessage"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &compiledOutput); err != nil {
		return nil, fmt.Errorf("parse solc output: %w", err)
	}
	var compilerErrors []string
	for _, diagnostic := range compiledOutput.Errors {
		if diagnostic.Severity == "error" {
			message := strings.TrimSpace(diagnostic.FormattedMessage)
			if message == "" {
				message = strings.TrimSpace(diagnostic.Message)
			}
			compilerErrors = append(compilerErrors, message)
		}
	}
	if len(compilerErrors) > 0 {
		return nil, fmt.Errorf("solc compilation failed: %s", strings.Join(compilerErrors, "\n"))
	}
	contracts := make([]compiledSolidityContract, 0)
	sourceNames := make(map[int]string, len(compiledOutput.Sources))
	for name, source := range compiledOutput.Sources {
		sourceNames[source.ID] = name
	}
	for sourceName, sourceContracts := range compiledOutput.Contracts {
		for contractName, artifact := range sourceContracts {
			if artifact.EVM.DeployedBytecode.Object == "" {
				continue
			}
			key := sourceName + ":" + contractName
			parsedABI, err := parseCombinedJSONABI(artifact.ABI)
			if err != nil {
				return nil, fmt.Errorf("parse ABI for %s: %w", key, err)
			}
			contracts = append(contracts, compiledSolidityContract{
				key: key, name: contractName, constructorBytecode: "0x" + artifact.EVM.Bytecode.Object,
				runtimeBytecode:  "0x" + artifact.EVM.DeployedBytecode.Object,
				runtimeSourceMap: artifact.EVM.DeployedBytecode.SourceMap,
				abi:              parsedABI, sourceNames: sourceNames,
				mutabilities:      parseABIMutabilities(artifact.ABI),
				functionLocations: solidityFunctionLocations(compiledOutput.Sources[sourceName].AST, contractName, sourceNames),
			})
		}
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].key < contracts[j].key })
	if len(contracts) == 0 {
		return nil, fmt.Errorf("solc produced no deployable contracts for %s", source)
	}
	return contracts, nil
}

func solidityFunctionLocations(ast solcASTNode, contractName string, sourceNames map[int]string) map[string]soliditySourceLocation {
	locations := make(map[string]soliditySourceLocation)
	for _, contract := range ast.Nodes {
		if contract.NodeType != "ContractDefinition" || contract.Name != contractName {
			continue
		}
		for _, node := range contract.Nodes {
			if node.NodeType != "FunctionDefinition" || node.FunctionSelector == "" {
				continue
			}
			location, ok := parseSoliditySourceLocation(node.Src, sourceNames)
			if ok {
				locations[strings.ToLower(node.FunctionSelector)] = location
			}
		}
	}
	return locations
}

func parseSoliditySourceLocation(src string, sourceNames map[int]string) (soliditySourceLocation, bool) {
	parts := strings.Split(src, ":")
	if len(parts) < 3 {
		return soliditySourceLocation{}, false
	}
	start, startErr := strconv.Atoi(parts[0])
	length, lengthErr := strconv.Atoi(parts[1])
	fileID, fileErr := strconv.Atoi(parts[2])
	file := sourceNames[fileID]
	if startErr != nil || lengthErr != nil || fileErr != nil || start < 0 || length < 0 || file == "" {
		return soliditySourceLocation{}, false
	}
	return soliditySourceLocation{File: file, Start: start, Length: length}, true
}

func buildRuntimeSourceMap(contract compiledSolidityContract) *solidityRuntimeSourceMap {
	if contract.runtimeSourceMap == "" {
		return nil
	}
	code, err := hex.DecodeString(strings.TrimPrefix(contract.runtimeBytecode, "0x"))
	if err != nil {
		return nil
	}
	pcs := solidityInstructionPCs(code)
	segments := strings.Split(contract.runtimeSourceMap, ";")
	locations := make([]solidityPCSourceLocation, 0, len(segments))
	start, length, fileID := -1, -1, -1
	for index, segment := range segments {
		if index >= len(pcs) {
			break
		}
		fields := strings.Split(segment, ":")
		if len(fields) > 0 && fields[0] != "" {
			start, _ = strconv.Atoi(fields[0])
		}
		if len(fields) > 1 && fields[1] != "" {
			length, _ = strconv.Atoi(fields[1])
		}
		if len(fields) > 2 && fields[2] != "" {
			fileID, _ = strconv.Atoi(fields[2])
		}
		file := contract.sourceNames[fileID]
		if start < 0 || length < 0 || file == "" {
			continue
		}
		locations = append(locations, solidityPCSourceLocation{
			PC: pcs[index], File: file, Start: start, Length: length,
		})
	}
	if len(locations) == 0 {
		return nil
	}
	return &solidityRuntimeSourceMap{Locations: locations}
}

func solidityInstructionPCs(code []byte) []uint64 {
	pcs := make([]uint64, 0, len(code))
	for pc := 0; pc < len(code); {
		pcs = append(pcs, uint64(pc))
		op := code[pc]
		pc++
		if op >= 0x60 && op <= 0x7f {
			pc += int(op - 0x5f)
			if pc > len(code) {
				pc = len(code)
			}
		}
	}
	return pcs
}

func parseCombinedJSONABI(raw json.RawMessage) (*ethabi.ABI, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, fmt.Errorf("missing ABI")
	}
	if raw[0] == '"' {
		var encoded string
		if err := json.Unmarshal(raw, &encoded); err != nil {
			return nil, err
		}
		raw = []byte(encoded)
	}
	return ethabi.NewABIFromReader(bytes.NewReader(raw))
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

func resolveSolidityMethod(contractABI *ethabi.ABI, requested string) (*ethabi.Method, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = "run"
	}
	var matches []*ethabi.Method
	for _, method := range contractABI.Methods {
		if method.Sig() == requested || (!strings.Contains(requested, "(") && method.Name == requested) {
			matches = append(matches, method)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		signatures := make([]string, len(matches))
		for i, method := range matches {
			signatures[i] = method.Sig()
		}
		sort.Strings(signatures)
		return nil, fmt.Errorf("function %q is overloaded; use a canonical signature: %s", requested, strings.Join(signatures, ", "))
	}
	return nil, fmt.Errorf("function %q not found in contract ABI", requested)
}

func buildConstructorData(contract compiledSolidityContract, argString string) (string, error) {
	if contract.constructorBytecode == "" || contract.constructorBytecode == "0x" {
		return "", fmt.Errorf("contract %s has no deployable constructor bytecode", contract.name)
	}
	arguments := constructorInputs(contract.abi)
	values, err := parseABIArguments(arguments, argString)
	if err != nil {
		return "", fmt.Errorf("encode constructor arguments: %w", err)
	}
	encoded, err := arguments.Encode(values)
	if err != nil {
		return "", fmt.Errorf("encode constructor arguments: %w", err)
	}
	return contract.constructorBytecode + fmt.Sprintf("%x", encoded), nil
}

func parseABIArguments(arguments *ethabi.Type, argString string) ([]interface{}, error) {
	valuesText := []string{}
	if argString != "" {
		valuesText = splitArgs(argString)
	}
	elements := arguments.TupleElems()
	if len(elements) != len(valuesText) {
		return nil, fmt.Errorf("argument count mismatch: expected %d, got %d", len(elements), len(valuesText))
	}
	values := make([]interface{}, len(elements))
	for i, argument := range elements {
		value, err := parseArg(valuesText[i], argument.Elem)
		if err != nil {
			return nil, fmt.Errorf("argument %d (%s): %w", i, argument.Elem.String(), err)
		}
		values[i] = value
	}
	return values, nil
}

func solidityParameters(arguments *ethabi.Type) []solidityParameterOutput {
	elements := arguments.TupleElems()
	parameters := make([]solidityParameterOutput, len(elements))
	for i, argument := range elements {
		parameters[i] = solidityParameterOutput{Name: argument.Name, Type: argument.Elem.String()}
	}
	return parameters
}

func constructorInputs(contractABI *ethabi.ABI) *ethabi.Type {
	if contractABI.Constructor != nil {
		return contractABI.Constructor.Inputs
	}
	t, _ := ethabi.NewType("tuple()")
	return t
}

func solidityMutability(method *ethabi.Method, declared string) string {
	if declared != "" {
		return declared
	}
	if method.Const {
		return "view"
	}
	return "nonpayable"
}

type abiJSONArgument struct {
	Type       string            `json:"type"`
	Components []abiJSONArgument `json:"components"`
}

func parseABIMutabilities(raw json.RawMessage) map[string]string {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil
	}
	if raw[0] == '"' {
		var encoded string
		if json.Unmarshal(raw, &encoded) != nil {
			return nil
		}
		raw = []byte(encoded)
	}
	var fields []struct {
		Type            string            `json:"type"`
		Name            string            `json:"name"`
		StateMutability string            `json:"stateMutability"`
		Inputs          []abiJSONArgument `json:"inputs"`
	}
	if json.Unmarshal(raw, &fields) != nil {
		return nil
	}
	result := make(map[string]string)
	for _, field := range fields {
		if field.Type != "function" && field.Type != "" {
			continue
		}
		parts := make([]string, len(field.Inputs))
		for index, input := range field.Inputs {
			parts[index] = canonicalABIJSONType(input)
		}
		result[field.Name+"("+strings.Join(parts, ",")+")"] = field.StateMutability
	}
	return result
}

func canonicalABIJSONType(argument abiJSONArgument) string {
	if !strings.HasPrefix(argument.Type, "tuple") {
		return argument.Type
	}
	parts := make([]string, len(argument.Components))
	for index, component := range argument.Components {
		parts[index] = canonicalABIJSONType(component)
	}
	return "(" + strings.Join(parts, ",") + ")" + strings.TrimPrefix(argument.Type, "tuple")
}

func solidityCompilerVersion(ctx context.Context, executable string, prefixArgs []string) string {
	args := append([]string{}, prefixArgs...)
	args = append(args, "--version")
	command := exec.CommandContext(ctx, executable, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "Version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		}
	}
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		return strings.TrimSpace(lines[len(lines)-1])
	}
	return "unknown"
}

func classifySolidityError(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "solc compilation"), strings.Contains(message, "parse solc output"):
		return "COMPILATION_FAILED"
	case strings.Contains(message, "contract "), strings.Contains(message, "function "):
		return "SELECTION_FAILED"
	case strings.Contains(message, "argument"):
		return "ARGUMENT_ERROR"
	default:
		return "SOLIDITY_RUN_FAILED"
	}
}

func writeSolidityError(out io.Writer, code string, err error) error {
	result := solidityErrorOutput{SchemaVersion: solidityProtocolVersion}
	result.Error.Code = code
	result.Error.Message = err.Error()
	return json.NewEncoder(out).Encode(result)
}

func writeSolidityRunOutput(out io.Writer, result solidityRunOutput, flags *solidityRunFlags) error {
	if flags.format == "evidence-json" {
		if result.Evidence == nil {
			return fmt.Errorf("missing Solidity evidence document")
		}
		jsonResult := solidityEvidenceJSONOutput{
			EvidenceDocument: *result.Evidence,
			Source:           result.Source, Contract: result.Contract, Function: result.Function, Compiler: result.Compiler,
		}
		return json.NewEncoder(out).Encode(jsonResult)
	}
	if flags.format == "summary-json" {
		jsonResult := solidityRunSummaryJSONOutput{
			SchemaVersion: agentSummarySchemaVersion,
			Source:        result.Source, Contract: result.Contract, Function: result.Function,
			Compiler: result.Compiler, DurationMS: result.DurationMS,
		}
		execution := summarizeExecution(result.Execution)
		jsonResult.Execution = &execution
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(jsonResult)
	}
	if flags.format == "json" {
		if !flags.trace {
			result.Execution.Trace = nil
		}
		jsonResult := solidityRunJSONOutput{
			SchemaVersion: result.SchemaVersion,
			Source:        result.Source, Contract: result.Contract, Function: result.Function,
			Compiler: result.Compiler, DurationMS: result.DurationMS, Execution: result.Execution,
			SourceMap: result.SourceMap,
		}
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(jsonResult)
	}
	if _, err := fmt.Fprintf(out, "EchoEVM Solidity run — %s:%s %s\n", result.Source, result.Contract, result.Function); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "status=%s return=%s gas=%d storage=%d\n", result.Execution.Status, result.Execution.ReturnData, result.Execution.GasUsed, len(result.Execution.Storage)); err != nil {
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
