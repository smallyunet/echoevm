package differential

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

func normalizeRequest(req Request) (Request, error) {
	if req.Fork == "" {
		req.Fork = ForkCancun
	}
	if !strings.EqualFold(req.Fork, ForkCancun) {
		return Request{}, fmt.Errorf("unsupported fork %q: only Cancun is supported", req.Fork)
	}
	req.Fork = ForkCancun
	if req.GasLimit == 0 {
		req.GasLimit = DefaultGasLimit
	}
	if req.GasLimit > MaxGasLimit {
		return Request{}, fmt.Errorf("gas limit %d exceeds maximum %d", req.GasLimit, MaxGasLimit)
	}
	code, err := decodeHexField("bytecode", req.Bytecode)
	if err != nil {
		return Request{}, err
	}
	if len(code) == 0 {
		return Request{}, fmt.Errorf("bytecode must not be empty")
	}
	if len(code) > MaxBytecodeBytes {
		return Request{}, fmt.Errorf("bytecode is %d bytes; maximum is %d", len(code), MaxBytecodeBytes)
	}
	input, err := decodeHexField("calldata", req.Calldata)
	if err != nil {
		return Request{}, err
	}
	if len(input) > MaxCalldataBytes {
		return Request{}, fmt.Errorf("calldata is %d bytes; maximum is %d", len(input), MaxCalldataBytes)
	}
	req.Bytecode = "0x" + hex.EncodeToString(code)
	req.Calldata = "0x" + hex.EncodeToString(input)
	storage := make(map[string]string, len(req.InitialStorage))
	for rawKey, rawValue := range req.InitialStorage {
		key, err := parseHash("storage key", rawKey)
		if err != nil {
			return Request{}, err
		}
		value, err := parseHash("storage value", rawValue)
		if err != nil {
			return Request{}, err
		}
		storage[key.Hex()] = value.Hex()
	}
	req.InitialStorage = storage
	return req, nil
}

func decodeHexField(name, value string) ([]byte, error) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if len(value)%2 != 0 {
		return nil, fmt.Errorf("invalid %s hex: odd length", name)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s hex: %w", name, err)
	}
	return decoded, nil
}

func parseHash(name, value string) (common.Hash, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if len(trimmed)%2 != 0 {
		trimmed = "0" + trimmed
	}
	decoded, err := decodeHexField(name, trimmed)
	if err != nil {
		return common.Hash{}, err
	}
	if len(decoded) > common.HashLength {
		return common.Hash{}, fmt.Errorf("%s exceeds 32 bytes", name)
	}
	return common.BytesToHash(decoded), nil
}

func canonicalWord(value string) string {
	value = strings.TrimPrefix(strings.ToLower(value), "0x")
	if value == "" {
		return "0x0"
	}
	n := new(big.Int)
	if _, ok := n.SetString(value, 16); !ok {
		return "0x" + value
	}
	return "0x" + n.Text(16)
}

func canonicalStack(values []string) []string {
	if values == nil {
		return nil
	}
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = canonicalWord(value)
	}
	return out
}

func storageKeys(req Request, traces ...[]NormalizedStep) []common.Hash {
	set := make(map[common.Hash]struct{})
	for key := range req.InitialStorage {
		set[common.HexToHash(key)] = struct{}{}
	}
	for _, trace := range traces {
		for _, step := range trace {
			if (step.OpcodeName == "SLOAD" || step.OpcodeName == "SSTORE") && len(step.StackBefore) > 0 {
				set[common.HexToHash(step.StackBefore[len(step.StackBefore)-1])] = struct{}{}
			}
		}
	}
	keys := make([]common.Hash, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return strings.Compare(keys[i].Hex(), keys[j].Hex()) < 0 })
	return keys
}

func moduleVersion(path string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	if info.Main.Path == path {
		if info.Main.Version == "" || info.Main.Version == "(devel)" {
			return "devel"
		}
		return info.Main.Version
	}
	for _, dependency := range info.Deps {
		if dependency.Path == path {
			return dependency.Version
		}
	}
	return "unknown"
}
