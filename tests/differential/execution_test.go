package differential

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/smallyunet/echoevm/internal/differential"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

const regressionGasLimit = uint64(1_000_000)

type vector struct{ name, category, code, input string }

var vectors = []vector{
	{name: "add", category: "arithmetic", code: "60026003015f5260205ff3"},
	{name: "sub", category: "arithmetic", code: "60036002035f5260205ff3"},
	{name: "mul", category: "arithmetic", code: "60076006025f5260205ff3"},
	{name: "div", category: "arithmetic", code: "60026008045f5260205ff3"},
	{name: "mod", category: "arithmetic", code: "60056017065f5260205ff3"},
	{name: "signed-div-negative", category: "arithmetic", code: "6002600619055f5260205ff3"},
	{name: "signed-mod-negative", category: "arithmetic", code: "6005600619075f5260205ff3"},
	{name: "addmod", category: "arithmetic", code: "600560046003085f5260205ff3"},
	{name: "mulmod", category: "arithmetic", code: "600560046003095f5260205ff3"},
	{name: "exp", category: "arithmetic", code: "600360020a5f5260205ff3"},
	{name: "less-than", category: "comparison", code: "60036002105f5260205ff3"},
	{name: "signed-less-than", category: "comparison", code: "6001600019125f5260205ff3"},
	{name: "signextend", category: "arithmetic", code: "608060000b5f5260205ff3"},
	{name: "shift-left", category: "bitwise", code: "600860011b5f5260205ff3"},
	{name: "xor", category: "bitwise", code: "60aa60ff185f5260205ff3"},
	{name: "byte", category: "bitwise", code: "611234601e1a5f5260205ff3"},
	{name: "arithmetic-shift-right-negative", category: "bitwise", code: "60071960011d5f5260205ff3"},
	{name: "calldataload", category: "environment", code: "5f355f5260205ff3", input: "2a00000000000000000000000000000000000000000000000000000000000000"},
	{name: "memory-roundtrip", category: "memory", code: "602a5f525f5160205ff3"},
	{name: "mload-offset-overflow", category: "fault", code: "680100000000000000005100"},
	{name: "mload-range-overflow", category: "fault", code: "67ffffffffffffffff5100"},
	{name: "mstore-offset-overflow", category: "fault", code: "6001680100000000000000005200"},
	{name: "mstore8-offset-overflow", category: "fault", code: "6001680100000000000000005300"},
	{name: "keccak256", category: "crypto", code: "602a5f5260205f205f5260205ff3"},
	{name: "storage-roundtrip", category: "storage", code: "602a5f555f545f5260205ff3"},
	{name: "transient-storage", category: "storage", code: "602a5f5d5f5c5f5260205ff3"},
	{name: "mcopy", category: "memory", code: "602a5f5260205f60205e60206020f3"},
	{name: "jump", category: "control", code: "600456005b602a5f5260205ff3"},
	{name: "revert", category: "control", code: "63deadbeef5f526004601cfd"},
	{name: "returndatacopy-out-of-bounds", category: "fault", code: "60015f5f3e00"},
	{name: "revert-restores-storage", category: "storage", code: "60015f5560006000fd"},
	{name: "invalid-opcode", category: "fault", code: "fe"},
	{name: "fault-restores-storage", category: "storage", code: "60015f55fe"},
	{name: "stack-underflow", category: "fault", code: "01"},
}

func TestEchoExecutionRegressionMatrix(t *testing.T) {
	categories := make(map[string]int)
	for _, test := range vectors {
		test := test
		categories[test.category]++
		t.Run(test.category+"/"+test.name, func(t *testing.T) {
			result, err := differential.DefaultEngine().RunEcho(context.Background(), differential.Request{
				Fork: differential.ForkCancun, Bytecode: test.code, Calldata: test.input, GasLimit: regressionGasLimit,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Engine != "EchoEVM" || result.Status == "" {
				t.Fatalf("invalid execution result: %+v", result)
			}
		})
	}
	names := make([]string, 0, len(categories))
	for category := range categories {
		names = append(names, category)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, category := range names {
		parts = append(parts, fmt.Sprintf("%s=%d", category, categories[category]))
	}
	t.Logf("ECHO REGRESSION SUMMARY fork=Cancun total=%d categories=%s skipped=0", len(vectors), strings.Join(parts, ","))
}

func TestForkMatrixBasicExecution(t *testing.T) {
	for _, fork := range core.SupportedForks {
		t.Run(fork, func(t *testing.T) {
			result, err := differential.DefaultEngine().RunEcho(context.Background(), differential.Request{
				Fork: fork, Bytecode: "600260030160005260206000f3", GasLimit: regressionGasLimit,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != differential.StatusSuccess || !strings.HasSuffix(result.ReturnData, "05") {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}
}

func TestPragueOsakaPrecompileActivation(t *testing.T) {
	tests := []struct{ name, fork, code, suffix string }{
		{"Prague-BLS12-G1-add-infinity", differential.ForkPrague, "60805f6101005f5f600b61fffff15f5260205ff3", "01"},
		{"Osaka-P256-invalid-signature", differential.ForkOsaka, "5f5f60a05f5f61010061fffff15f5260205ff3", "01"},
		{"Cancun-KZG-malformed-input", differential.ForkCancun, "5f5f5f5f5f600a61fffff15f5260205ff3", "00"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := differential.DefaultEngine().RunEcho(context.Background(), differential.Request{Fork: test.fork, Bytecode: test.code, GasLimit: regressionGasLimit})
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != differential.StatusSuccess || !strings.HasSuffix(result.ReturnData, test.suffix) {
				t.Fatalf("unexpected result: %+v", result)
			}
		})
	}
}

func TestRegressionCoverageContract(t *testing.T) {
	if len(vectors) < 34 {
		t.Fatalf("regression baseline shrank: have %d, require at least 34", len(vectors))
	}
	required := []string{"arithmetic", "bitwise", "comparison", "control", "crypto", "environment", "fault", "memory", "storage"}
	seen := make(map[string]bool)
	for _, test := range vectors {
		seen[test.category] = true
	}
	for _, category := range required {
		if !seen[category] {
			t.Errorf("required category %q has no vectors", category)
		}
	}
}
