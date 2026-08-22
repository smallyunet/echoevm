package differential

import (
	"context"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/smallyunet/echoevm/internal/differential"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

const differentialGasLimit = uint64(1_000_000)

type vector struct {
	name     string
	category string
	code     string
	input    string
}

func TestNestedCreateOutcomeMatrixAgainstGeth(t *testing.T) {
	tests := []struct {
		name     string
		create2  bool
		initCode []byte
	}{
		{
			name:     "create success",
			initCode: []byte{0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3},
		},
		{
			name:     "create revert restores state and gas",
			initCode: []byte{0x60, 0x01, 0x5f, 0x55, 0x5f, 0x5f, 0xfd},
		},
		{
			name:     "create exceptional halt restores state and burns gas",
			initCode: []byte{0x60, 0x01, 0x5f, 0x55, 0xfe},
		},
		{
			name:     "create2 success",
			create2:  true,
			initCode: []byte{0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3},
		},
		{
			name:     "create2 revert restores state and gas",
			create2:  true,
			initCode: []byte{0x60, 0x01, 0x5f, 0x55, 0x5f, 0x5f, 0xfd},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := differential.Request{
				Fork:     differential.ForkCancun,
				Bytecode: createHarnessBytecode(t, tt.initCode, tt.create2),
				GasLimit: differentialGasLimit,
			}
			echo, err := (differential.EchoRunner{}).Run(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			geth, err := (differential.GethRunner{}).Run(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if echo.Status != geth.Status || echo.ReturnData != geth.ReturnData || echo.GasUsed != geth.GasUsed || !reflect.DeepEqual(echo.Storage, geth.Storage) {
				t.Fatalf("nested create outcome differs:\nEchoEVM status=%s return=%s gas=%d storage=%v\nGeth status=%s return=%s gas=%d storage=%v",
					echo.Status, echo.ReturnData, echo.GasUsed, echo.Storage,
					geth.Status, geth.ReturnData, geth.GasUsed, geth.Storage)
			}
		})
	}
}

func TestNestedCallOutcomeMatrixAgainstGeth(t *testing.T) {
	tests := []struct {
		name       string
		static     bool
		childCode  []byte
		wantResult byte
		wantState  string
	}{
		{
			name:       "call success commits state",
			childCode:  []byte{0x60, 0x01, 0x5f, 0x55, 0x00},
			wantResult: 1,
			wantState:  "0x01",
		},
		{
			name:      "call revert restores state and gas",
			childCode: []byte{0x60, 0x01, 0x5f, 0x55, 0x5f, 0x5f, 0xfd},
			wantState: "0x00",
		},
		{
			name:      "call exceptional halt restores state and burns gas",
			childCode: []byte{0x60, 0x01, 0x5f, 0x55, 0xfe},
			wantState: "0x00",
		},
		{
			name:      "staticcall rejects state write",
			static:    true,
			childCode: []byte{0x60, 0x01, 0x5f, 0x55, 0x00},
			wantState: "0x00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := differential.Request{
				Fork:     differential.ForkCancun,
				Bytecode: callHarnessBytecode(t, tt.childCode, tt.static),
				GasLimit: differentialGasLimit,
				InitialStorage: map[string]string{
					"0x00": "0x00",
				},
			}
			echo, err := (differential.EchoRunner{}).Run(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			geth, err := (differential.GethRunner{}).Run(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if echo.Status != geth.Status || echo.ReturnData != geth.ReturnData || echo.GasUsed != geth.GasUsed || !reflect.DeepEqual(echo.Storage, geth.Storage) {
				t.Fatalf("nested call outcome differs:\nEchoEVM status=%s return=%s gas=%d storage=%v\nGeth status=%s return=%s gas=%d storage=%v",
					echo.Status, echo.ReturnData, echo.GasUsed, echo.Storage,
					geth.Status, geth.ReturnData, geth.GasUsed, geth.Storage)
			}
			if !strings.HasSuffix(echo.ReturnData, fmt.Sprintf("%02x", tt.wantResult)) {
				t.Fatalf("call result = %s, want final byte %02x", echo.ReturnData, tt.wantResult)
			}
			key := "0x0000000000000000000000000000000000000000000000000000000000000000"
			if !strings.HasSuffix(echo.Storage[key], strings.TrimPrefix(tt.wantState, "0x")) {
				t.Fatalf("storage[0] = %s, want %s", echo.Storage[key], tt.wantState)
			}
		})
	}
}

func createHarnessBytecode(t *testing.T, initCode []byte, create2 bool) string {
	t.Helper()
	if len(initCode) > 255 {
		t.Fatalf("test initcode is too large: %d", len(initCode))
	}
	prefixLength := byte(17)
	if create2 {
		prefixLength = 19
	}
	code := []byte{
		0x60, byte(len(initCode)), 0x60, prefixLength, 0x5f, 0x39, // CODECOPY initcode to memory[0:]
	}
	if create2 {
		code = append(code, 0x60, 0x07) // salt
	}
	code = append(code, 0x60, byte(len(initCode)), 0x5f, 0x5f)
	if create2 {
		code = append(code, 0xf5)
	} else {
		code = append(code, 0xf0)
	}
	code = append(code, 0x5f, 0x52, 0x60, 0x20, 0x5f, 0xf3) // return created address or zero
	code = append(code, initCode...)
	return hex.EncodeToString(code)
}

func callHarnessBytecode(t *testing.T, childCode []byte, static bool) string {
	t.Helper()
	parentLabel := 5 + len(childCode)
	if parentLabel > 255 {
		t.Fatalf("test child code is too large: %d", len(childCode))
	}
	code := []byte{0x36, 0x15, 0x60, byte(parentLabel), 0x57} // calldata empty selects parent
	code = append(code, childCode...)
	code = append(code, 0x5b, 0x60, 0x01, 0x5f, 0x53) // parent: memory[0] = 1
	if static {
		// retLength, retOffset, argsLength, argsOffset, address, gas, STATICCALL
		code = append(code, 0x5f, 0x5f, 0x60, 0x01, 0x5f, 0x30, 0x61, 0xff, 0xff, 0xfa)
	} else {
		// retLength, retOffset, argsLength, argsOffset, value, address, gas, CALL
		code = append(code, 0x5f, 0x5f, 0x60, 0x01, 0x5f, 0x5f, 0x30, 0x61, 0xff, 0xff, 0xf1)
	}
	code = append(code, 0x5f, 0x52, 0x60, 0x20, 0x5f, 0xf3) // return call success flag
	return hex.EncodeToString(code)
}

// These vectors intentionally exercise small, independently diagnosable pieces
// of the Cancun VM. Geth is the oracle; a behavior change in EchoEVM must match
// the same return data, halt class, and persistent storage result in geth.
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

func TestCancunDifferentialAgainstGeth(t *testing.T) {
	engine := differential.DefaultEngine()
	categories := make(map[string]int)
	for _, test := range vectors {
		test := test
		categories[test.category]++
		t.Run(fmt.Sprintf("%s/%s", test.category, test.name), func(t *testing.T) {
			result, err := engine.Compare(context.Background(), differential.Request{
				Fork: differential.ForkCancun, Bytecode: test.code, Calldata: test.input, GasLimit: differentialGasLimit,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.Match {
				t.Fatalf("first divergence: %+v", result.FirstDivergence)
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
	t.Logf("DIFFERENTIAL SUMMARY fork=Cancun total=%d categories=%s skipped=0", len(vectors), strings.Join(parts, ","))
}

func TestForkMatrixBasicExecutionAgainstGeth(t *testing.T) {
	for _, fork := range core.SupportedForks {
		t.Run(fork, func(t *testing.T) {
			result, err := differential.DefaultEngine().Compare(context.Background(), differential.Request{
				Fork: fork, Bytecode: "600260030160005260206000f3", GasLimit: differentialGasLimit,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.Match {
				t.Fatalf("first divergence: %+v", result.FirstDivergence)
			}
		})
	}
}

func TestForkOpcodeActivationAgainstGeth(t *testing.T) {
	tests := []struct {
		name, before, active, code string
	}{
		{name: "revert", before: differential.ForkHomestead, active: differential.ForkByzantium, code: "60006000fd"},
		{name: "shift", before: differential.ForkByzantium, active: differential.ForkConstantinople, code: "600160011b00"},
		{name: "chainid", before: differential.ForkConstantinople, active: differential.ForkIstanbul, code: "4660005260206000f3"},
		{name: "basefee", before: differential.ForkBerlin, active: differential.ForkLondon, code: "4860005260206000f3"},
		{name: "push0", before: differential.ForkParis, active: differential.ForkShanghai, code: "5f00"},
		{name: "tstore", before: differential.ForkShanghai, active: differential.ForkCancun, code: "600160005d00"},
		{name: "clz", before: differential.ForkPrague, active: differential.ForkOsaka, code: "60001e60005260206000f3"},
	}
	for _, test := range tests {
		for _, fork := range []string{test.before, test.active} {
			t.Run(test.name+"/"+fork, func(t *testing.T) {
				result, err := differential.DefaultEngine().Compare(context.Background(), differential.Request{
					Fork: fork, Bytecode: test.code, GasLimit: differentialGasLimit,
				})
				if err != nil {
					t.Fatal(err)
				}
				if !result.Match {
					t.Fatalf("first divergence: %+v", result.FirstDivergence)
				}
			})
		}
	}
}

func TestPragueOsakaPrecompilesAgainstGeth(t *testing.T) {
	tests := []struct {
		name string
		fork string
		code string
	}{
		{
			name: "Prague-BLS12-G1-add-infinity",
			fork: differential.ForkPrague,
			// CALL 0x0b with 256 zero bytes and return the success flag.
			code: "60805f6101005f5f600b61fffff15f5260205ff3",
		},
		{
			name: "Osaka-P256-invalid-signature",
			fork: differential.ForkOsaka,
			// CALL 0x100 with the 160-byte all-zero verification input.
			code: "5f5f60a05f5f61010061fffff15f5260205ff3",
		},
		{
			name: "Cancun-KZG-malformed-input",
			fork: differential.ForkCancun,
			// A malformed point-evaluation input must fail the child call.
			code: "5f5f5f5f5f600a61fffff15f5260205ff3",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := differential.DefaultEngine().Compare(context.Background(), differential.Request{
				Fork: test.fork, Bytecode: test.code, GasLimit: differentialGasLimit,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.Match {
				t.Fatalf("first divergence: %+v", result.FirstDivergence)
			}
		})
	}
}

func TestDifferentialCoverageContract(t *testing.T) {
	const minimumVectors = 34
	if len(vectors) < minimumVectors {
		t.Fatalf("differential baseline shrank: have %d vectors, require at least %d", len(vectors), minimumVectors)
	}
	requiredCategories := []string{"arithmetic", "bitwise", "comparison", "control", "crypto", "environment", "fault", "memory", "storage"}
	seen := make(map[string]bool)
	for _, test := range vectors {
		if test.name == "" || test.category == "" || test.code == "" {
			t.Fatalf("differential vector is missing required metadata: %+v", test)
		}
		seen[test.category] = true
	}
	for _, category := range requiredCategories {
		if !seen[category] {
			t.Errorf("required differential category %q has no vectors", category)
		}
	}
}
