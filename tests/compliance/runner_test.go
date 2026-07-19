package compliance

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
)

type fixtureFile struct {
	Meta  fixtureMeta   `json:"_meta"`
	Cases []fixtureCase `json:"cases"`
}

type fixtureMeta struct {
	SourceRepository string `json:"sourceRepository"`
	SourceCommit     string `json:"sourceCommit"`
	SourceFile       string `json:"sourceFile"`
	Fork             string `json:"fork"`
}

type fixtureCase struct {
	Name        string                    `json:"name"`
	Category    string                    `json:"category"`
	Pre         map[string]fixtureAccount `json:"pre"`
	Transaction fixtureTransaction        `json:"transaction"`
	Post        map[string]fixtureAccount `json:"post"`
	Block       fixtureBlock              `json:"block"`
	Return      string                    `json:"return"`
	Reverted    bool                      `json:"reverted"`
	Error       string                    `json:"error"`
}

type fixtureAccount struct {
	Balance string            `json:"balance"`
	Code    string            `json:"code"`
	Nonce   string            `json:"nonce"`
	Storage map[string]string `json:"storage"`
}

type fixtureTransaction struct {
	Data      string `json:"data"`
	GasLimit  string `json:"gasLimit"`
	GasPrice  string `json:"gasPrice"`
	Nonce     string `json:"nonce"`
	SecretKey string `json:"secretKey"`
	Sender    string `json:"sender"`
	To        string `json:"to"`
	Value     string `json:"value"`
}

type fixtureBlock struct {
	Coinbase   string `json:"coinbase"`
	Difficulty string `json:"difficulty"`
	GasLimit   string `json:"gasLimit"`
	Number     string `json:"number"`
	Timestamp  string `json:"timestamp"`
	BaseFee    string `json:"baseFee"`
}

func parseBig(t *testing.T, value string) *big.Int {
	t.Helper()
	if value == "" {
		return new(big.Int)
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(value, "0x"), 16)
	if !ok {
		t.Fatalf("invalid hexadecimal integer %q", value)
	}
	return n
}

func parseBytes(t *testing.T, value string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
	if err != nil {
		t.Fatalf("invalid hexadecimal bytes %q: %v", value, err)
	}
	return b
}

func loadState(t *testing.T, accounts map[string]fixtureAccount) *core.MemoryStateDB {
	t.Helper()
	statedb := core.NewMemoryStateDB()
	for address, account := range accounts {
		addr := common.HexToAddress(address)
		statedb.CreateAccount(addr)
		statedb.AddBalance(addr, parseBig(t, account.Balance))
		statedb.SetNonce(addr, parseBig(t, account.Nonce).Uint64())
		statedb.SetCode(addr, parseBytes(t, account.Code))
		for key, value := range account.Storage {
			statedb.SetState(addr, common.HexToHash(key), common.HexToHash(value))
		}
	}
	return statedb
}

func runFixture(t *testing.T, test fixtureCase) {
	t.Helper()
	statedb := loadState(t, test.Pre)

	privateKey, err := crypto.ToECDSA(parseBytes(t, test.Transaction.SecretKey))
	if err != nil {
		t.Fatalf("invalid transaction secret key: %v", err)
	}
	sender := crypto.PubkeyToAddress(privateKey.PublicKey)
	if expectedSender := common.HexToAddress(test.Transaction.Sender); sender != expectedSender {
		t.Fatalf("derived sender %s does not match fixture sender %s", sender, expectedSender)
	}

	to := common.HexToAddress(test.Transaction.To)
	tx := types.NewTransaction(
		parseBig(t, test.Transaction.Nonce).Uint64(),
		to,
		parseBig(t, test.Transaction.Value),
		parseBig(t, test.Transaction.GasLimit).Uint64(),
		parseBig(t, test.Transaction.GasPrice),
		parseBytes(t, test.Transaction.Data),
	)
	ctx := &vm.BlockContext{
		BlockNumber: parseBig(t, test.Block.Number),
		Timestamp:   parseBig(t, test.Block.Timestamp).Uint64(),
		Coinbase:    common.HexToAddress(test.Block.Coinbase),
		GasLimit:    parseBig(t, test.Block.GasLimit).Uint64(),
		Difficulty:  parseBig(t, test.Block.Difficulty),
		BaseFee:     parseBig(t, test.Block.BaseFee),
	}

	ret, _, reverted, executionErr := vm.ApplyTransactionWithContext(statedb, tx, sender, ctx)
	gotError := ""
	if executionErr != nil {
		gotError = executionErr.Error()
	}
	if got := gotError; got != test.Error {
		t.Fatalf("execution error mismatch: want %q, got %q", test.Error, got)
	}
	if reverted != test.Reverted {
		t.Fatalf("reverted mismatch: want %t, got %t", test.Reverted, reverted)
	}
	if expected := parseBytes(t, test.Return); !bytes.Equal(ret, expected) {
		t.Fatalf("return data mismatch: want 0x%x, got 0x%x", expected, ret)
	}

	for address, account := range test.Post {
		addr := common.HexToAddress(address)
		if account.Balance != "" && statedb.GetBalance(addr).Cmp(parseBig(t, account.Balance)) != 0 {
			t.Errorf("balance mismatch for %s: want %s, got 0x%x", addr, account.Balance, statedb.GetBalance(addr))
		}
		if account.Nonce != "" && statedb.GetNonce(addr) != parseBig(t, account.Nonce).Uint64() {
			t.Errorf("nonce mismatch for %s: want %s, got 0x%x", addr, account.Nonce, statedb.GetNonce(addr))
		}
		if account.Code != "" && !bytes.Equal(statedb.GetCode(addr), parseBytes(t, account.Code)) {
			t.Errorf("code mismatch for %s", addr)
		}
		for key, value := range account.Storage {
			want := common.HexToHash(value)
			if got := statedb.GetState(addr, common.HexToHash(key)); got != want {
				t.Errorf("storage mismatch for %s at %s: want %s, got %s", addr, key, want, got)
			}
		}
	}
}

func TestCompliance(t *testing.T) {
	fixturePaths, err := filepath.Glob(filepath.Join("fixtures", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixturePaths) == 0 {
		t.Fatal("no compliance fixture files found")
	}

	executed := 0
	categories := make(map[string]int)
	forks := make(map[string]int)
	for _, path := range fixturePaths {
		path := path
		t.Run(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), func(t *testing.T) {
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var fixture fixtureFile
			if err := json.Unmarshal(contents, &fixture); err != nil {
				t.Fatal(err)
			}
			if fixture.Meta.SourceRepository == "" || fixture.Meta.SourceCommit == "" || fixture.Meta.SourceFile == "" || fixture.Meta.Fork == "" {
				t.Fatal("fixture source metadata is required")
			}
			if len(fixture.Cases) == 0 {
				t.Fatal("fixture file contains no cases")
			}
			for _, test := range fixture.Cases {
				test := test
				if test.Name == "" || test.Category == "" {
					t.Fatalf("fixture case is missing name or category: %+v", test)
				}
				executed++
				categories[test.Category]++
				forks[fixture.Meta.Fork]++
				t.Run(test.Name, func(t *testing.T) {
					runFixture(t, test)
				})
			}
		})
	}
	if executed == 0 {
		t.Fatal("no compliance cases executed")
	}
	if executed < 9 {
		t.Fatalf("official compliance baseline shrank: executed %d cases, require at least 9", executed)
	}
	t.Logf("COMPLIANCE SUMMARY official=%d categories=%s forks=%s skipped=0", executed, formatCounts(categories), formatCounts(forks))
}

func formatCounts(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, ",")
}
