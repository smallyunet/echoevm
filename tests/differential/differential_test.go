package differential

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
)

const differentialGasLimit = uint64(1_000_000)

var contractAddress = common.BytesToAddress([]byte("contract"))

type vector struct {
	name     string
	category string
	code     string
	input    string
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
	{name: "shift-left", category: "bitwise", code: "600860011b5f5260205ff3"},
	{name: "xor", category: "bitwise", code: "60aa60ff185f5260205ff3"},
	{name: "calldataload", category: "environment", code: "5f355f5260205ff3", input: "2a00000000000000000000000000000000000000000000000000000000000000"},
	{name: "memory-roundtrip", category: "memory", code: "602a5f525f5160205ff3"},
	{name: "keccak256", category: "crypto", code: "602a5f5260205f205f5260205ff3"},
	{name: "storage-roundtrip", category: "storage", code: "602a5f555f545f5260205ff3"},
	{name: "transient-storage", category: "storage", code: "602a5f5d5f5c5f5260205ff3"},
	{name: "mcopy", category: "memory", code: "602a5f5260205f60205e60206020f3"},
	{name: "jump", category: "control", code: "600456005b602a5f5260205ff3"},
	{name: "revert", category: "control", code: "63deadbeef5f526004601cfd"},
	{name: "invalid-opcode", category: "fault", code: "fe"},
	{name: "stack-underflow", category: "fault", code: "01"},
}

type result struct {
	output  []byte
	status  string
	storage common.Hash
	gasUsed uint64
}

func decodeHex(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
	if err != nil {
		t.Fatalf("invalid test vector hex %q: %v", value, err)
	}
	return decoded
}

func runEchoEVM(t *testing.T, test vector) result {
	t.Helper()
	state := core.NewMemoryStateDB()
	state.PrepareTransaction()
	state.AddAddressToAccessList(contractAddress)
	intr := vm.NewWithCallData(decodeHex(t, test.code), decodeHex(t, test.input), state, contractAddress)
	intr.SetGas(differentialGasLimit)
	intr.Run()

	status := "success"
	if intr.Err() != nil {
		status = "fault"
	} else if intr.IsReverted() {
		status = "revert"
	}
	return result{
		output:  intr.ReturnedCode(),
		status:  status,
		storage: state.GetState(contractAddress, common.Hash{}),
		gasUsed: differentialGasLimit - intr.Gas(),
	}
}

func runGeth(t *testing.T, test vector) result {
	t.Helper()
	var gasUsed uint64
	output, state, err := runtime.Execute(
		decodeHex(t, test.code),
		decodeHex(t, test.input),
		&runtime.Config{
			GasLimit: differentialGasLimit,
			EVMConfig: gethvm.Config{Tracer: &tracing.Hooks{
				OnExit: func(depth int, _ []byte, used uint64, _ error, _ bool) {
					if depth == 0 {
						gasUsed = used
					}
				},
			}},
		},
	)
	status := "success"
	if errors.Is(err, gethvm.ErrExecutionReverted) {
		status = "revert"
	} else if err != nil {
		status = "fault"
	}
	return result{
		output:  output,
		status:  status,
		storage: state.GetState(contractAddress, common.Hash{}),
		gasUsed: gasUsed,
	}
}

func TestCancunDifferentialAgainstGeth(t *testing.T) {
	categories := make(map[string]int)
	for _, test := range vectors {
		test := test
		categories[test.category]++
		t.Run(fmt.Sprintf("%s/%s", test.category, test.name), func(t *testing.T) {
			echoResult := runEchoEVM(t, test)
			gethResult := runGeth(t, test)
			if echoResult.status != gethResult.status {
				t.Fatalf("halt mismatch: echoevm=%s geth=%s", echoResult.status, gethResult.status)
			}
			if !bytes.Equal(echoResult.output, gethResult.output) {
				t.Fatalf("return mismatch: echoevm=0x%x geth=0x%x", echoResult.output, gethResult.output)
			}
			if echoResult.storage != gethResult.storage {
				t.Fatalf("storage[0] mismatch: echoevm=%s geth=%s", echoResult.storage, gethResult.storage)
			}
			if echoResult.gasUsed != gethResult.gasUsed {
				t.Fatalf("gas mismatch: echoevm=%d geth=%d", echoResult.gasUsed, gethResult.gasUsed)
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

func TestDifferentialCoverageContract(t *testing.T) {
	const minimumVectors = 15
	if len(vectors) < minimumVectors {
		t.Fatalf("differential baseline shrank: have %d vectors, require at least %d", len(vectors), minimumVectors)
	}
	requiredCategories := []string{"arithmetic", "bitwise", "control", "crypto", "environment", "fault", "memory", "storage"}
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
