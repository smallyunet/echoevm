package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
)

const fakeSolcOutput = `{"contracts":{"Example.sol":{"Answer":{"abi":[{"inputs":[{"name":"left","type":"uint256"},{"name":"right","type":"uint256"}],"name":"add","outputs":[{"name":"","type":"uint256"}],"stateMutability":"pure","type":"function"}],"evm":{"bytecode":{"object":"67602a5f5260205ff360005260086018f3"},"deployedBytecode":{"object":"602a5f5260205ff3"}}}}}}`

func TestSolidityRunCompilesExecutesAndDiffs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake solc fixture uses a POSIX shell")
	}
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "Example.sol")
	if err := os.WriteFile(source, []byte("contract Answer {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	compiler := writeFakeSolc(t, tempDir, successfulFakeSolcBody(""))
	flags := &solidityRunFlags{
		contract: "Answer", function: "add", args: "2,40", solc: compiler,
		gas: 100_000, format: "json", diff: true,
	}
	var output bytes.Buffer
	if err := runSolidity(t.Context(), &output, source, flags); err != nil {
		t.Fatalf("run Solidity: %v", err)
	}
	var result solidityRunJSONOutput
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("decode output: %v\n%s", err, output.String())
	}
	if result.Contract != "Answer" || result.Function != "add(uint256,uint256)" {
		t.Fatalf("unexpected selection: contract=%s function=%s", result.Contract, result.Function)
	}
	if result.Comparison == nil || !result.Comparison.Match {
		t.Fatalf("expected a matching differential result: %+v", result.Comparison)
	}
	const fortyTwo = "0x000000000000000000000000000000000000000000000000000000000000002a"
	if result.Execution.ReturnData != fortyTwo {
		t.Fatalf("return data = %s, want %s", result.Execution.ReturnData, fortyTwo)
	}
	if result.SchemaVersion != solidityProtocolVersion {
		t.Fatalf("schema version = %d", result.SchemaVersion)
	}
	if result.Execution.Trace != nil || result.Comparison.Geth.Trace != nil {
		t.Fatal("JSON output included traces without --trace")
	}
	if strings.Contains(output.String(), `"bytecode"`) || strings.Contains(output.String(), `"initCode"`) {
		t.Fatalf("editor JSON leaked full execution request: %s", output.String())
	}
}

func TestSolidityRunSummaryUsesSeparateDeploymentGas(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake solc fixture uses a POSIX shell")
	}
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "Example.sol")
	if err := os.WriteFile(source, []byte("contract Answer {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	compiler := writeFakeSolc(t, tempDir, successfulFakeSolcBody(""))
	flags := &solidityRunFlags{
		contract: "Answer", function: "add", args: "2,40", solc: compiler,
		gas: 10_000, deployGas: 100_000, format: "summary-json", diff: true,
	}
	var output bytes.Buffer
	if err := runSolidity(t.Context(), &output, source, flags); err != nil {
		t.Fatalf("run Solidity summary: %v", err)
	}
	var result solidityRunSummaryJSONOutput
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("decode summary: %v\n%s", err, output.String())
	}
	if result.SchemaVersion != agentSummarySchemaVersion || result.Comparison == nil || !result.Comparison.Match {
		t.Fatalf("unexpected summary: %+v", result)
	}
	if result.Comparison.Request.GasLimit != 10_000 || result.Comparison.Request.DeployGasLimit != 100_000 {
		t.Fatalf("gas limits not preserved: %+v", result.Comparison.Request)
	}
	if result.Execution != nil || result.Comparison.EchoEVM.TraceSteps == 0 || strings.Contains(output.String(), "stackBefore") || strings.Contains(output.String(), `"bytecode":`) {
		t.Fatalf("summary leaked verbose execution fields: %s", output.String())
	}
}

func TestSolidityRunTraceAndCompilerArguments(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake solc fixture uses a POSIX shell")
	}
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "Example.sol")
	if err := os.WriteFile(source, []byte("contract Answer {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	argsFile := filepath.Join(tempDir, "args.txt")
	inputFile := filepath.Join(tempDir, "input.json")
	script := successfulFakeSolcBody("printf '%s\\n' \"$@\" > " + shellSingleQuote(argsFile) + "\ncat > " + shellSingleQuote(inputFile) + "\n")
	compiler := writeFakeSolc(t, tempDir, script)
	flags := &solidityRunFlags{
		function: "add(uint256,uint256)", args: "2,40", solc: compiler,
		gas: 100_000, format: "json", trace: true, optimize: true,
		optimizerRuns: 100, viaIR: true, remappings: []string{"@example/=lib/example/"},
	}
	var output bytes.Buffer
	if err := runSolidity(t.Context(), &output, source, flags); err != nil {
		t.Fatalf("run Solidity: %v", err)
	}
	var result solidityRunJSONOutput
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Execution.Trace) == 0 {
		t.Fatal("--trace did not include opcode steps")
	}
	compilerArgs, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	joined := string(compilerArgs)
	for _, expected := range []string{"--standard-json", "--base-path"} {
		if !strings.Contains(joined, expected) {
			t.Errorf("compiler arguments missing %q: %s", expected, joined)
		}
	}
	compilerInput, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatal(err)
	}
	input := string(compilerInput)
	for _, expected := range []string{
		`"evmVersion":"cancun"`, `"enabled":true`, `"runs":100`, `"viaIR":true`,
		`"remappings":["@example/=lib/example/"]`,
	} {
		if !strings.Contains(input, expected) {
			t.Fatalf("standard JSON input missing %s: %s", expected, input)
		}
	}
}

func TestSolidityInspectListsContractsAndFunctions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake solc fixture uses a POSIX shell")
	}
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "Example.sol")
	if err := os.WriteFile(source, []byte("contract Answer {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	compiler := writeFakeSolc(t, tempDir, successfulFakeSolcBody(""))
	var output bytes.Buffer
	err := runSolidityInspect(t.Context(), &output, source, &solidityRunFlags{
		solc: compiler, gas: 100_000, format: "json",
	})
	if err != nil {
		t.Fatalf("inspect Solidity: %v", err)
	}
	var result solidityInspectOutput
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != 1 || result.Compiler.Version != "0.8.30+commit.test" {
		t.Fatalf("unexpected protocol metadata: %+v", result)
	}
	if len(result.Contracts) != 1 || len(result.Contracts[0].Functions) != 1 {
		t.Fatalf("unexpected inspection result: %+v", result.Contracts)
	}
	if result.Contracts[0].Functions[0].Signature != "add(uint256,uint256)" {
		t.Fatalf("unexpected function: %+v", result.Contracts[0].Functions[0])
	}
}

func TestSolidityRunReportsCompilerFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake solc fixture uses a POSIX shell")
	}
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "Broken.sol")
	if err := os.WriteFile(source, []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	compiler := writeFakeSolc(t, tempDir, "echo 'ParserError: expected declaration' >&2\nexit 1\n")
	err := runSolidity(t.Context(), &bytes.Buffer{}, source, &solidityRunFlags{solc: compiler, gas: 100_000, format: "text"})
	if err == nil || !strings.Contains(err.Error(), "ParserError: expected declaration") {
		t.Fatalf("unexpected compiler error: %v", err)
	}
}

func TestSolidityRunReportsStandardJSONDiagnostic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake solc fixture uses a POSIX shell")
	}
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "Broken.sol")
	if err := os.WriteFile(source, []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	body := "if [ \"$1\" = \"--version\" ]; then echo 'Version: 0.8.30'; exit 0; fi\n" +
		"cat >/dev/null\n" +
		"printf '%s' '{\"errors\":[{\"severity\":\"error\",\"formattedMessage\":\"ParserError: expected declaration\"}]}'\n"
	compiler := writeFakeSolc(t, tempDir, body)
	err := runSolidity(t.Context(), &bytes.Buffer{}, source, &solidityRunFlags{solc: compiler, gas: 100_000, format: "text"})
	if err == nil || !strings.Contains(err.Error(), "ParserError: expected declaration") {
		t.Fatalf("unexpected standard JSON compiler error: %v", err)
	}
}

func TestWriteSolidityErrorUsesVersionedEnvelope(t *testing.T) {
	var output bytes.Buffer
	err := writeSolidityError(&output, "ARGUMENT_ERROR", os.ErrInvalid)
	if err != nil {
		t.Fatal(err)
	}
	var result solidityErrorOutput
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != solidityProtocolVersion || result.Error.Code != "ARGUMENT_ERROR" {
		t.Fatalf("unexpected error envelope: %+v", result)
	}
}

func TestResolveSolidityMethodRequiresSignatureForOverload(t *testing.T) {
	contractABI, err := gethabi.JSON(strings.NewReader(`[
      {"type":"function","name":"read","inputs":[{"name":"x","type":"uint256"}],"outputs":[]},
      {"type":"function","name":"read","inputs":[{"name":"x","type":"address"}],"outputs":[]}
    ]`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolveSolidityMethod(contractABI, "read"); err == nil || !strings.Contains(err.Error(), "overloaded") {
		t.Fatalf("expected overload error, got %v", err)
	}
	method, err := resolveSolidityMethod(contractABI, "read(address)")
	if err != nil {
		t.Fatal(err)
	}
	if method.Sig != "read(address)" {
		t.Fatalf("resolved %s", method.Sig)
	}
}

func TestBuildConstructorDataEncodesArguments(t *testing.T) {
	contractABI, err := gethabi.JSON(strings.NewReader(`[
      {"type":"constructor","inputs":[{"name":"initialValue","type":"uint256"}]}
    ]`))
	if err != nil {
		t.Fatal(err)
	}
	data, err := buildConstructorData(compiledSolidityContract{
		name: "Stateful", constructorBytecode: "0x6000", abi: contractABI,
	}, "7")
	if err != nil {
		t.Fatal(err)
	}
	const expected = "0x60000000000000000000000000000000000000000000000000000000000000000007"
	if data != expected {
		t.Fatalf("constructor data = %s, want %s", data, expected)
	}
}

func TestSelectCompiledContractRejectsAmbiguousName(t *testing.T) {
	contracts := []compiledSolidityContract{
		{key: "a.sol:Token", name: "Token"},
		{key: "b.sol:Token", name: "Token"},
	}
	if _, err := selectCompiledContract(contracts, "Token"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous contract error, got %v", err)
	}
	selected, err := selectCompiledContract(contracts, "b.sol:Token")
	if err != nil || selected.key != "b.sol:Token" {
		t.Fatalf("source-qualified selection failed: selected=%+v err=%v", selected, err)
	}
}

func writeFakeSolc(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "fake-solc")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func successfulFakeSolcBody(beforeCompile string) string {
	return "if [ \"$1\" = \"--version\" ]; then\n" +
		"  printf '%s\\n' 'solc, the solidity compiler' 'Version: 0.8.30+commit.test'\n" +
		"  exit 0\n" +
		"fi\n" + beforeCompile + "printf '%s' '" + fakeSolcOutput + "'\n"
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
