package compliance

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"github.com/smallyunet/echoevm/internal/evm/vm"
	"github.com/smallyunet/echoevm/internal/trie"
)

// Test structures matching ethereum/tests JSON format
type StateTest struct {
	Env         Env                    `json:"env"`
	Pre         map[string]Account     `json:"pre"`
	Transaction Transaction            `json:"transaction"`
	Post        map[string][]PostState `json:"post"`
}

type PostState struct {
	Hash    common.Hash        `json:"hash"`
	Indexes map[string]int     `json:"indexes"`
	Logs    common.Hash        `json:"logs"`
	State   map[string]Account `json:"state"`
	TxBytes hexutil.Bytes      `json:"txbytes"`
}

type Env struct {
	CurrentCoinbase   common.Address `json:"currentCoinbase"`
	CurrentDifficulty string         `json:"currentDifficulty"`
	CurrentGasLimit   string         `json:"currentGasLimit"`
	CurrentNumber     string         `json:"currentNumber"`
	CurrentTimestamp  string         `json:"currentTimestamp"`
	CurrentBaseFee    string         `json:"currentBaseFee"`
}

type Account struct {
	Balance string            `json:"balance"`
	Code    hexutil.Bytes     `json:"code"`
	Nonce   string            `json:"nonce"`
	Storage map[string]string `json:"storage"`
}

type Transaction struct {
	Data      []hexutil.Bytes `json:"data"`
	GasLimit  []string        `json:"gasLimit"`
	GasPrice  string          `json:"gasPrice"`
	Nonce     string          `json:"nonce"`
	SecretKey common.Hash     `json:"secretKey"`
	To        *common.Address `json:"to"`
	Sender    *common.Address `json:"sender"` // Sometimes provided, otherwise derived
	Value     []string        `json:"value"`
}

func toBig(hex string) *big.Int {
	if hex == "" {
		return new(big.Int)
	}
	n, _ := new(big.Int).SetString(strings.TrimPrefix(hex, "0x"), 16)
	return n
}

func toUint64(hex string) uint64 {
	if hex == "" {
		return 0
	}
	n, _ := new(big.Int).SetString(strings.TrimPrefix(hex, "0x"), 16)
	return n.Uint64()
}

// mockDB implements trie.Database for testing
type mockDB struct {
	data map[common.Hash][]byte
}

func newMockDB() *mockDB {
	return &mockDB{
		data: make(map[common.Hash][]byte),
	}
}

func (db *mockDB) Node(hash common.Hash) ([]byte, error) {
	if val, ok := db.data[hash]; ok {
		return val, nil
	}
	return nil, nil
}

func (db *mockDB) Put(hash common.Hash, val []byte) error {
	db.data[hash] = val
	return nil
}

// MakeTrieState creates a TrieStateBackend from the pre-state
func MakeTrieState(pre map[string]Account) (*core.TrieStateBackend, common.Hash, error) {
	db := newMockDB()
	accTrie, err := trie.New(common.Hash{}, db)
	if err != nil {
		return nil, common.Hash{}, err
	}

	for addrStr, accObj := range pre {
		addr := common.HexToAddress(addrStr)

		// 1. Setup Storage Trie
		storageTrie, err := trie.New(common.Hash{}, db)
		if err != nil {
			return nil, common.Hash{}, err
		}
		for k, v := range accObj.Storage {
			key := common.HexToHash(k)
			val := common.HexToHash(v)
			valBytes, _ := rlp.EncodeToBytes(val)
			storageTrie.Update(crypto.Keccak256(key[:]), valBytes)
		}
		storageRoot, err := storageTrie.Commit()
		if err != nil {
			return nil, common.Hash{}, err
		}

		// 2. Setup Account
		trieAcc := core.TrieAccount{
			Nonce:    toUint64(accObj.Nonce),
			Balance:  toBig(accObj.Balance),
			Root:     storageRoot,
			CodeHash: crypto.Keccak256(accObj.Code),
		}

		// Store code in DB if present
		if len(accObj.Code) > 0 {
			db.Put(common.BytesToHash(trieAcc.CodeHash), accObj.Code)
		}

		accBytes, _ := rlp.EncodeToBytes(trieAcc)
		accTrie.Update(crypto.Keccak256(addr[:]), accBytes)
	}

	root, err := accTrie.Commit()
	if err != nil {
		return nil, common.Hash{}, err
	}

	backend, err := core.NewTrieStateBackend(root, db)
	return backend, root, err
}

// RunTest executes a single test case
func RunTest(t *testing.T, name string, test StateTest) {
	// 1. Setup StateDB
	backend, _, err := MakeTrieState(test.Pre)
	if err != nil {
		t.Fatalf("Failed to create trie state: %v", err)
	}
	statedb := core.NewMemoryStateDB()
	statedb.SetBackend(backend)

	// 2. Setup VM & Transaction
	// Note: GeneralStateTests can have multiple data/value/gasLimit indexes.
	// We usually iterate them. For simplicity here, we take the first one (index 0).

	data := test.Transaction.Data[0]
	gasLimit := toUint64(test.Transaction.GasLimit[0])
	value := toBig(test.Transaction.Value[0])
	gasPrice := toBig(test.Transaction.GasPrice)
	nonce := toUint64(test.Transaction.Nonce)

	var tx *types.Transaction
	if test.Transaction.To != nil {
		tx = types.NewTransaction(nonce, *test.Transaction.To, value, gasLimit, gasPrice, data)
	} else {
		tx = types.NewContractCreation(nonce, value, gasLimit, gasPrice, data)
	}

	// Determine Sender
	// Standard test sender
	sender := common.HexToAddress("0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b")

	// Setup Block Context
	ctx := &vm.BlockContext{
		BlockNumber: toBig(test.Env.CurrentNumber),
		Timestamp:   toUint64(test.Env.CurrentTimestamp),
		Coinbase:    test.Env.CurrentCoinbase,
		GasLimit:    toUint64(test.Env.CurrentGasLimit),
		Difficulty:  toBig(test.Env.CurrentDifficulty),
		BaseFee:     toBig(test.Env.CurrentBaseFee),
	}

	// 3. Run
	_, _, _, err = vm.ApplyTransactionWithContext(statedb, tx, sender, ctx)

	if err != nil {
		// Some tests expect failure.
		// t.Logf("Execution error: %v", err)
		_ = err
	}

	// 4. Verify Post State
	for fork, postStates := range test.Post {
		// For now, we only verify if we find a post state that matches our execution (index 0)
		// and we assume our VM behaves like the fork specified.
		// Since we don't support fork configuration yet, this is a best-effort verification.
		for i, postState := range postStates {
			if postState.Indexes["data"] != 0 || postState.Indexes["gas"] != 0 || postState.Indexes["value"] != 0 {
				continue
			}

			t.Logf("Verifying post state for fork: %s, index: %d", fork, i)
			for addrStr, expectedAcc := range postState.State {
				addr := common.HexToAddress(addrStr)

				// Verify Balance
				expectedBalance := toBig(expectedAcc.Balance)
				actualBalance := statedb.GetBalance(addr)
				if expectedBalance.Cmp(actualBalance) != 0 {
					t.Errorf("[%s] Balance mismatch for %s: expected %v, got %v", fork, addrStr, expectedBalance, actualBalance)
				}

				// Verify Nonce
				expectedNonce := toUint64(expectedAcc.Nonce)
				actualNonce := statedb.GetNonce(addr)
				if expectedNonce != actualNonce {
					t.Errorf("[%s] Nonce mismatch for %s: expected %v, got %v", fork, addrStr, expectedNonce, actualNonce)
				}

				// Verify Code
				expectedCode := expectedAcc.Code
				actualCode := statedb.GetCode(addr)
				if len(expectedCode) != len(actualCode) { // Simple length check first
					t.Errorf("[%s] Code length mismatch for %s: expected %d, got %d", fork, addrStr, len(expectedCode), len(actualCode))
				} else {
					// Deep comparison if needed, or just rely on length for now if large
					for i := range expectedCode {
						if expectedCode[i] != actualCode[i] {
							t.Errorf("[%s] Code mismatch for %s at byte %d", fork, addrStr, i)
							break
						}
					}
				}

				// Verify Storage
				for keyStr, valStr := range expectedAcc.Storage {
					key := common.HexToHash(keyStr)
					expectedVal := common.HexToHash(valStr)
					actualVal := statedb.GetState(addr, key)
					if expectedVal != actualVal {
						t.Errorf("[%s] Storage mismatch for %s at %s: expected %s, got %s", fork, addrStr, keyStr, expectedVal.Hex(), actualVal.Hex())
					}
				}
			}
		}
	}
}

// TestCompliance runs all JSON tests in a directory
func TestCompliance(t *testing.T) {
	fixturesDir := "../fixtures/GeneralStateTests/VMTests" // Start with VMTests as they are simpler
	if _, err := os.Stat(fixturesDir); os.IsNotExist(err) {
		t.Skip("Fixtures not found, skipping compliance tests")
	}

	err := filepath.Walk(fixturesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		t.Run(filepath.Base(path), func(t *testing.T) {
			file, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			var tests map[string]StateTest
			if err := json.Unmarshal(file, &tests); err != nil {
				t.Fatal(err)
			}

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					RunTest(t, name, test)
				})
			}
		})
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}
