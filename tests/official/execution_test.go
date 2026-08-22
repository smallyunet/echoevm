package official

import (
	"bytes"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/types"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
)

type stateFixture struct {
	Env  fixtureEnv                          `json:"env"`
	Pre  types.GenesisAlloc                  `json:"pre"`
	Post map[string][]fixturePostTransaction `json:"post"`
}

type fixtureEnv struct {
	Coinbase      common.Address `json:"currentCoinbase"`
	GasLimit      string         `json:"currentGasLimit"`
	Number        string         `json:"currentNumber"`
	Timestamp     string         `json:"currentTimestamp"`
	Difficulty    string         `json:"currentDifficulty"`
	Random        string         `json:"currentRandom"`
	BaseFee       string         `json:"currentBaseFee"`
	ExcessBlobGas string         `json:"currentExcessBlobGas"`
}

type fixturePostTransaction struct {
	TxBytes         string             `json:"txbytes"`
	State           types.GenesisAlloc `json:"state"`
	ExpectException string             `json:"expectException"`
}

func TestOsakaOfficialExecutionCorpus(t *testing.T) {
	root := os.Getenv("ECHOEVM_OFFICIAL_FIXTURES")
	if root == "" {
		t.Skip("full official fixtures are opt-in; run make test-official-fixtures")
	}
	corpus := discoverOsakaStateFixtures(t, root)
	if len(corpus) < 180 {
		t.Fatalf("current-fork corpus shrank unexpectedly: files=%d want>=180", len(corpus))
	}
	validated := 0
	rejected := 0
	for _, relativePath := range corpus {
		relativePath := relativePath
		t.Run(strings.TrimSuffix(filepath.Base(relativePath), ".json"), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
			if err != nil {
				t.Fatalf("read official fixture %s: %v", relativePath, err)
			}
			var fixtures map[string]stateFixture
			if err := json.Unmarshal(data, &fixtures); err != nil {
				t.Fatalf("decode official fixture %s: %v", relativePath, err)
			}
			names := make([]string, 0, len(fixtures))
			for name := range fixtures {
				if !strings.HasPrefix(name, "_") {
					names = append(names, name)
				}
			}
			sort.Strings(names)
			if len(names) == 0 {
				t.Fatalf("official fixture %s contains no executable cases", relativePath)
			}
			for _, name := range names {
				fixture := fixtures[name]
				for index, post := range fixture.Post[core.ForkOsaka] {
					wasRejected := runOfficialStateCase(t, name, index, fixture, post)
					validated++
					if wasRejected {
						rejected++
					}
				}
			}
		})
	}
	if validated < 3000 {
		t.Fatalf("current-fork corpus shrank unexpectedly: transactions=%d want>=3000", validated)
	}
	t.Logf("OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files=%d transactions=%d accepted=%d rejected=%d fork=Osaka skipped=0", len(corpus), validated, validated-rejected, rejected)
}

func discoverOsakaStateFixtures(t *testing.T, root string) []string {
	t.Helper()
	var result []string
	// Execute every Prague- and Osaka-authored fixture under Osaka rules. Older
	// historical suites that happen to include an Osaka post-state remain in the
	// release-wide inventory audit and are not mixed into this current-mainnet
	// conformance claim.
	for _, authoredFork := range []string{"prague", "osaka"} {
		stateRoot := filepath.Join(root, "state_tests", "for_osaka", authoredFork)
		err := filepath.WalkDir(stateRoot, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".json" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !bytes.Contains(data, []byte(`"Osaka"`)) {
				return nil
			}
			var fixtures map[string]struct {
				Post map[string]json.RawMessage `json:"post"`
			}
			if err := json.Unmarshal(data, &fixtures); err != nil {
				return err
			}
			for _, fixture := range fixtures {
				if _, ok := fixture.Post[core.ForkOsaka]; ok {
					relative, err := filepath.Rel(root, path)
					if err != nil {
						return err
					}
					result = append(result, filepath.ToSlash(relative))
					break
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("discover current-fork fixtures: %v", err)
		}
	}
	sort.Strings(result)
	return result
}

func runOfficialStateCase(t *testing.T, name string, index int, fixture stateFixture, post fixturePostTransaction) bool {
	t.Helper()
	if post.TxBytes == "" || post.TxBytes == "0x" {
		if post.ExpectException != "" {
			return true
		}
		t.Fatalf("%s[%d] has neither txbytes nor expectException", name, index)
		return false
	}
	var tx types.Transaction
	txBytes, err := common.ParseHexOrString(post.TxBytes)
	if err != nil {
		if post.ExpectException != "" {
			return true
		}
		t.Fatalf("%s[%d] parse txbytes: %v", name, index, err)
	}
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		if post.ExpectException != "" {
			return true
		}
		t.Fatalf("%s[%d] decode txbytes: %v", name, index, err)
	}
	signer := types.HomesteadSigner()
	if tx.Protected() || tx.Type() != types.LegacyTxType {
		signer = types.LatestSignerForChainID(tx.ChainId())
	}
	sender, err := types.Sender(signer, &tx)
	if err != nil {
		if post.ExpectException != "" {
			return true
		}
		t.Fatalf("%s[%d] recover sender: %v", name, index, err)
	}
	state := core.NewMemoryStateDB()
	loadGenesisAlloc(state, fixture.Pre)
	chainConfig, err := core.ChainConfigForFork(core.ForkOsaka)
	if err != nil {
		t.Fatalf("%s[%d] configure Osaka: %v", name, index, err)
	}
	blockNumber := mustFixtureUint64(t, name, "currentNumber", fixture.Env.Number)
	timestamp := mustFixtureUint64(t, name, "currentTimestamp", fixture.Env.Timestamp)
	gasLimit := mustFixtureUint64(t, name, "currentGasLimit", fixture.Env.GasLimit)
	ctx := &vm.BlockContext{
		BlockNumber: new(big.Int).SetUint64(blockNumber),
		Timestamp:   timestamp,
		Coinbase:    fixture.Env.Coinbase,
		GasLimit:    gasLimit,
		ChainID:     big.NewInt(1),
		ChainConfig: chainConfig,
	}
	if fixture.Env.BaseFee != "" {
		ctx.BaseFee = mustFixtureBig(t, name, "currentBaseFee", fixture.Env.BaseFee)
	}
	if fixture.Env.Difficulty != "" {
		ctx.Difficulty = mustFixtureBig(t, name, "currentDifficulty", fixture.Env.Difficulty)
	}
	if fixture.Env.Random != "" {
		ctx.Random = mustFixtureBig(t, name, "currentRandom", fixture.Env.Random)
	}
	if fixture.Env.ExcessBlobGas != "" {
		excessBlobGas := mustFixtureUint64(t, name, "currentExcessBlobGas", fixture.Env.ExcessBlobGas)
		ctx.BlobBaseFee = core.CalcBlobFee(excessBlobGas)
	}
	_, _, _, applyErr := vm.ApplyTransactionWithContext(state, &tx, sender, ctx)
	if post.ExpectException != "" {
		if applyErr == nil {
			t.Fatalf("%s[%d] expected %s, transaction was accepted", name, index, post.ExpectException)
		}
		return true
	}
	// An EVM exceptional halt is a validly included transaction and is reflected
	// by the post-state. Consensus-invalid transactions are represented by
	// expectException above, so post-state comparison is the authoritative check.
	assertGenesisAlloc(t, name, index, state, post.State)
	return false
}

func loadGenesisAlloc(state *core.MemoryStateDB, alloc types.GenesisAlloc) {
	for address, account := range alloc {
		state.CreateAccount(address)
		if account.Balance != nil {
			state.AddBalance(address, account.Balance)
		}
		state.SetNonce(address, account.Nonce)
		state.SetCode(address, account.Code)
		for key, value := range account.Storage {
			state.InitState(address, key, value)
		}
	}
	state.ClearJournal()
}

func assertGenesisAlloc(t *testing.T, name string, index int, state *core.MemoryStateDB, want types.GenesisAlloc) {
	t.Helper()
	for address, account := range want {
		if got := state.GetBalance(address); account.Balance == nil || got.Cmp(account.Balance) != 0 {
			t.Errorf("%s[%d] %s balance=%s want=%v", name, index, address, got, account.Balance)
		}
		if got := state.GetNonce(address); got != account.Nonce {
			t.Errorf("%s[%d] %s nonce=%d want=%d", name, index, address, got, account.Nonce)
		}
		if got := state.GetCode(address); !bytes.Equal(got, account.Code) {
			t.Errorf("%s[%d] %s code=%x want=%x", name, index, address, got, account.Code)
		}
		actualStorage := make(map[common.Hash]common.Hash)
		state.ForEachStorage(address, func(key, value common.Hash) bool {
			if value != (common.Hash{}) {
				actualStorage[key] = value
			}
			return true
		})
		for key, value := range account.Storage {
			if value == (common.Hash{}) {
				delete(account.Storage, key)
			}
		}
		if !reflect.DeepEqual(actualStorage, account.Storage) {
			t.Errorf("%s[%d] %s storage=%v want=%v", name, index, address, actualStorage, account.Storage)
		}
	}
}

func mustFixtureUint64(t *testing.T, name, field, value string) uint64 {
	t.Helper()
	parsed, err := strconv.ParseUint(strings.TrimPrefix(value, "0x"), 16, 64)
	if err != nil {
		t.Fatalf("%s parse %s=%q: %v", name, field, value, err)
	}
	return parsed
}

func mustFixtureBig(t *testing.T, name, field, value string) *big.Int {
	t.Helper()
	parsed, ok := new(big.Int).SetString(strings.TrimPrefix(value, "0x"), 16)
	if !ok {
		t.Fatalf("%s parse %s=%q", name, field, value)
	}
	return parsed
}
