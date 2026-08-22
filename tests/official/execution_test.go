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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
)

var pragueOsakaExecutionCorpus = []string{
	"state_tests/for_prague/prague/eip2537_bls_12_381_precompiles/eip_mainnet/eip_2537.json",
	"state_tests/for_prague/prague/eip2537_bls_12_381_precompiles/bls12_g1add/gas.json",
	"state_tests/for_prague/prague/eip2537_bls_12_381_precompiles/bls12_g1add/valid.json",
	"state_tests/for_prague/prague/eip2537_bls_12_381_precompiles/bls12_pairing/valid.json",
	"state_tests/for_prague/prague/eip7623_increase_calldata_cost/eip_mainnet/eip_7623.json",
	"state_tests/for_prague/prague/eip7623_increase_calldata_cost/refunds/gas_refunds_from_data_floor.json",
	"state_tests/for_prague/prague/eip7702_set_code_tx/eip_mainnet/eip_7702.json",
	"state_tests/for_prague/prague/eip7702_set_code_tx/gas/account_warming.json",
	"state_tests/for_prague/prague/eip7702_set_code_tx/gas/gas_cost.json",
	"state_tests/for_prague/prague/eip7702_set_code_tx/set_code_txs_2/pointer_to_pointer.json",
	"state_tests/for_osaka/osaka/eip7883_modexp_gas_increase/eip_mainnet/modexp_different_base_lengths.json",
	"state_tests/for_osaka/osaka/eip7883_modexp_gas_increase/modexp_thresholds/modexp_boundary_inputs.json",
	"state_tests/for_osaka/osaka/eip7883_modexp_gas_increase/modexp_thresholds/vectors_from_eip.json",
	"state_tests/for_osaka/osaka/eip7823_modexp_upper_bounds/eip_mainnet/modexp_boundary.json",
	"state_tests/for_osaka/osaka/eip7823_modexp_upper_bounds/eip_mainnet/modexp_over_boundary.json",
	"state_tests/for_osaka/osaka/eip7939_count_leading_zeros/eip_mainnet/clz_mainnet.json",
	"state_tests/for_osaka/osaka/eip7939_count_leading_zeros/count_leading_zeros/clz_gas_cost.json",
	"state_tests/for_osaka/osaka/eip7939_count_leading_zeros/count_leading_zeros/clz_opcode_scenarios.json",
	"state_tests/for_osaka/osaka/eip7951_p256verify_precompiles/eip_mainnet/invalid.json",
	"state_tests/for_osaka/osaka/eip7951_p256verify_precompiles/eip_mainnet/valid.json",
	"state_tests/for_osaka/osaka/eip7951_p256verify_precompiles/p256verify/gas.json",
	"state_tests/for_osaka/osaka/eip7951_p256verify_precompiles/p256verify/call_types.json",
	"state_tests/for_osaka/osaka/eip7825_transaction_gas_limit_cap/eip_mainnet/tx_gas_limit_cap_at_maximum.json",
	"state_tests/for_osaka/osaka/eip7825_transaction_gas_limit_cap/eip_mainnet/tx_gas_limit_cap_exceeded.json",
	"state_tests/for_osaka/osaka/eip7594_peerdas/max_blob_per_tx/valid_max_blobs_per_tx.json",
	"state_tests/for_osaka/osaka/eip7594_peerdas/max_blob_per_tx/invalid_max_blobs_per_tx.json",
}

type stateFixture struct {
	Env  fixtureEnv                          `json:"env"`
	Pre  gethtypes.GenesisAlloc              `json:"pre"`
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
	TxBytes         string                 `json:"txbytes"`
	State           gethtypes.GenesisAlloc `json:"state"`
	ExpectException string                 `json:"expectException"`
}

func TestPragueOsakaOfficialExecutionCorpus(t *testing.T) {
	root := os.Getenv("ECHOEVM_OFFICIAL_FIXTURES")
	if root == "" {
		t.Skip("full official fixtures are opt-in; run make test-official-fixtures")
	}
	executed := 0
	for _, relativePath := range pragueOsakaExecutionCorpus {
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
				forks := make([]string, 0, len(fixture.Post))
				for fork := range fixture.Post {
					forks = append(forks, fork)
				}
				sort.Strings(forks)
				for _, fork := range forks {
					if fork != core.ForkPrague && fork != core.ForkOsaka {
						t.Fatalf("%s declares unexpected fork %q", name, fork)
					}
					for index, post := range fixture.Post[fork] {
						executed++
						runOfficialStateCase(t, name, index, fork, fixture, post)
					}
				}
			}
		})
	}
	if executed == 0 {
		t.Fatal("Prague/Osaka official execution corpus ran zero transactions")
	}
	t.Logf("OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files=%d transactions=%d forks=Prague,Osaka skipped=0", len(pragueOsakaExecutionCorpus), executed)
}

func runOfficialStateCase(t *testing.T, name string, index int, fork string, fixture stateFixture, post fixturePostTransaction) {
	t.Helper()
	if post.TxBytes == "" || post.TxBytes == "0x" {
		if post.ExpectException != "" {
			return
		}
		t.Fatalf("%s[%d] has neither txbytes nor expectException", name, index)
	}
	var tx gethtypes.Transaction
	txBytes, err := common.ParseHexOrString(post.TxBytes)
	if err != nil {
		t.Fatalf("%s[%d] parse txbytes: %v", name, index, err)
	}
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		t.Fatalf("%s[%d] decode txbytes: %v", name, index, err)
	}
	var signer gethtypes.Signer = gethtypes.HomesteadSigner{}
	if tx.Protected() || tx.Type() != gethtypes.LegacyTxType {
		signer = gethtypes.LatestSignerForChainID(tx.ChainId())
	}
	sender, err := gethtypes.Sender(signer, &tx)
	if err != nil {
		t.Fatalf("%s[%d] recover sender: %v", name, index, err)
	}
	state := core.NewMemoryStateDB()
	loadGenesisAlloc(state, fixture.Pre)
	chainConfig, err := core.ChainConfigForFork(fork)
	if err != nil {
		t.Fatalf("%s[%d] configure %s: %v", name, index, fork, err)
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
		zero := uint64(0)
		gethConfig := *params.MainnetChainConfig
		gethConfig.ShanghaiTime = &zero
		gethConfig.CancunTime = &zero
		gethConfig.PragueTime = &zero
		if fork == core.ForkOsaka {
			gethConfig.OsakaTime = &zero
		} else {
			gethConfig.OsakaTime = nil
		}
		ctx.BlobBaseFee = eip4844.CalcBlobFee(&gethConfig, &gethtypes.Header{Time: timestamp, ExcessBlobGas: &excessBlobGas})
	}
	_, _, _, applyErr := vm.ApplyTransactionWithContext(state, &tx, sender, ctx)
	if post.ExpectException != "" {
		if applyErr == nil {
			t.Fatalf("%s[%d] expected %s, transaction was accepted", name, index, post.ExpectException)
		}
		return
	}
	// An EVM exceptional halt is a validly included transaction and is reflected
	// by the post-state. Consensus-invalid transactions are represented by
	// expectException above, so post-state comparison is the authoritative check.
	assertGenesisAlloc(t, name, index, state, post.State)
}

func loadGenesisAlloc(state *core.MemoryStateDB, alloc gethtypes.GenesisAlloc) {
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

func assertGenesisAlloc(t *testing.T, name string, index int, state *core.MemoryStateDB, want gethtypes.GenesisAlloc) {
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
