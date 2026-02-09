package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/smallyunet/echoevm/internal/trie"
)

// TrieStateBackend implements StateBackend using a Merkle Patricia Trie.
type TrieStateBackend struct {
	db   trie.Database
	trie *trie.Trie
}

// NewTrieStateBackend creates a new TrieStateBackend.
func NewTrieStateBackend(root common.Hash, db trie.Database) (*TrieStateBackend, error) {
	t, err := trie.New(root, db)
	if err != nil {
		return nil, err
	}
	return &TrieStateBackend{
		db:   db,
		trie: t,
	}, nil
}

// TrieAccount is the RLP encoding of an account in the state trie.
type TrieAccount struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash
	CodeHash []byte
}

func (ts *TrieStateBackend) GetAccount(addr common.Address) (*Account, error) {
	// Account trie uses Keccak256(addr) as key (Secure Trie)
	key := crypto.Keccak256(addr[:])
	enc := ts.trie.Get(key)
	if len(enc) == 0 {
		return nil, nil // Not found
	}
	var ta TrieAccount
	if err := rlp.DecodeBytes(enc, &ta); err != nil {
		return nil, err
	}

	acc := NewAccount()
	acc.Nonce = ta.Nonce
	acc.Balance = ta.Balance
	acc.CodeHash = ta.CodeHash
	acc.Root = ta.Root

	// Load code if code hash is present and not empty hash
	if len(ta.CodeHash) > 0 && common.BytesToHash(ta.CodeHash) != crypto.Keccak256Hash(nil) {
		code, err := ts.db.Node(common.BytesToHash(ta.CodeHash))
		if err == nil {
			acc.Code = code
		}
		// If code missing, we just don't set it (maybe error?)
	}

	return acc, nil
}

func (ts *TrieStateBackend) GetStorage(addr common.Address, key common.Hash) (common.Hash, error) {
	// Account key
	accKey := crypto.Keccak256(addr[:])
	enc := ts.trie.Get(accKey)
	if len(enc) == 0 {
		return common.Hash{}, nil
	}
	var ta TrieAccount
	if err := rlp.DecodeBytes(enc, &ta); err != nil {
		return common.Hash{}, err
	}

	if ta.Root == (common.Hash{}) {
		return common.Hash{}, nil
	}

	storageTrie, err := trie.New(ta.Root, ts.db)
	if err != nil {
		return common.Hash{}, err
	}

	// Storage key
	storeKey := crypto.Keccak256(key[:])
	valEnc := storageTrie.Get(storeKey)
	if len(valEnc) == 0 {
		return common.Hash{}, nil
	}

	var val common.Hash
	if err := rlp.DecodeBytes(valEnc, &val); err != nil {
		return common.Hash{}, err
	}
	return val, nil
}
