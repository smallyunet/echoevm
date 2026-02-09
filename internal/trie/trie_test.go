package trie

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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
		return nil, nil // Not found
	}
	return val, nil
}

func (db *mockDB) Put(hash common.Hash, val []byte) error {
	db.data[hash] = val
	return nil
}

func TestTrie_Insert_Get(t *testing.T) {
	db := newMockDB()
	trie, err := New(common.Hash{}, db)
	if err != nil {
		t.Fatalf("failed to create trie: %v", err)
	}

	key := []byte("foo")
	val := []byte("bar")

	trie.Update(key, val)

	got := trie.Get(key)
	if !bytes.Equal(got, val) {
		t.Errorf("expected %s, got %s", val, got)
	}

	// Test update
	val2 := []byte("baz")
	trie.Update(key, val2)
	got = trie.Get(key)
	if !bytes.Equal(got, val2) {
		t.Errorf("expected %s, got %s", val2, got)
	}
}

func TestTrie_Delete(t *testing.T) {
	db := newMockDB()
	trie, err := New(common.Hash{}, db)
	if err != nil {
		t.Fatalf("failed to create trie: %v", err)
	}

	key := []byte("foo")
	val := []byte("bar")

	trie.Update(key, val)
	trie.Delete(key)

	got := trie.Get(key)
	if got != nil {
		t.Errorf("expected nil, got %s", got)
	}
}

func TestTrie_Hash(t *testing.T) {
	db := newMockDB()
	trie, err := New(common.Hash{}, db)
	if err != nil {
		t.Fatalf("failed to create trie: %v", err)
	}

	trie.Update([]byte("foo"), []byte("bar"))
	root := trie.Hash()

	if (root == common.Hash{}) {
		t.Error("expected non-empty root hash")
	}
}

func TestTrie_MultipleInserts(t *testing.T) {
	db := newMockDB()
	trie, err := New(common.Hash{}, db)
	if err != nil {
		t.Fatalf("failed to create trie: %v", err)
	}

	kv := map[string]string{
		"do":    "verb",
		"dog":   "puppy",
		"doge":  "coin",
		"horse": "stallion",
	}

	for k, v := range kv {
		trie.Update([]byte(k), []byte(v))
	}

	for k, v := range kv {
		got := trie.Get([]byte(k))
		if !bytes.Equal(got, []byte(v)) {
			t.Errorf("key %s: expected %s, got %s", k, v, got)
		}
	}
}
