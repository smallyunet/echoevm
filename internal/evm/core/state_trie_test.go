package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/smallyunet/echoevm/internal/trie"
)

type mockDB struct {
	data map[common.Hash][]byte
}

func newMockDB() *mockDB {
	return &mockDB{
		data: make(map[common.Hash][]byte),
	}
}

func (db *mockDB) Node(hash common.Hash) ([]byte, error) {
	val, ok := db.data[hash]
	if !ok {
		return nil, nil
	}
	return val, nil
}

func (db *mockDB) Put(hash common.Hash, val []byte) error {
	db.data[hash] = val
	return nil
}

func TestTrieStateBackend(t *testing.T) {
	db := newMockDB()

	// 1. Setup Storage Trie
	storageTrie, err := trie.New(common.Hash{}, db)
	if err != nil {
		t.Fatal(err)
	}
	storageKey := common.HexToHash("0x11") // Key (slot)
	storageVal := common.HexToHash("0x99") // Value

	// MPT storage values are RLP encoded
	valBytes, _ := rlp.EncodeToBytes(storageVal)

	// Backend hashes the key.
	storageTrie.Update(crypto.Keccak256(storageKey[:]), valBytes)
	storageRoot, err := storageTrie.Commit()
	if err != nil {
		t.Fatal(err)
	}

	// 2. Setup Account Trie
	accTrie, err := trie.New(common.Hash{}, db)
	if err != nil {
		t.Fatal(err)
	}

	addr := common.HexToAddress("0x1234")
	acc := TrieAccount{
		Nonce:    1,
		Balance:  big.NewInt(100),
		Root:     storageRoot,
		CodeHash: crypto.Keccak256(nil),
	}
	accBytes, _ := rlp.EncodeToBytes(acc)
	accTrie.Update(crypto.Keccak256(addr[:]), accBytes)

	root, err := accTrie.Commit()
	if err != nil {
		t.Fatal(err)
	}

	// 3. Test TrieStateBackend
	backend, err := NewTrieStateBackend(root, db)
	if err != nil {
		t.Fatal(err)
	}

	// Test GetAccount
	gotAcc, err := backend.GetAccount(addr)
	if err != nil {
		t.Fatal(err)
	}
	if gotAcc == nil {
		t.Fatal("account not found")
	}
	if gotAcc.Nonce != 1 {
		t.Errorf("expected nonce 1, got %d", gotAcc.Nonce)
	}
	if gotAcc.Balance.Cmp(big.NewInt(100)) != 0 {
		t.Errorf("expected balance 100, got %v", gotAcc.Balance)
	}

	// Test GetStorage
	gotVal, err := backend.GetStorage(addr, storageKey)
	if err != nil {
		t.Fatal(err)
	}
	if gotVal != storageVal {
		t.Errorf("expected storage val %x, got %x", storageVal, gotVal)
	}
}
