package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Account struct {
	Nonce           uint64
	Balance         *big.Int
	CodeHash        []byte // using []byte to store hash for simplicity, though common.Hash is [32]byte
	Code            []byte
	Storage         map[common.Hash]common.Hash
	OriginalStorage map[common.Hash]common.Hash // EIP-2200
	Suicided        bool
}

func NewAccount() *Account {
	return &Account{
		Balance:         new(big.Int),
		Storage:         make(map[common.Hash]common.Hash),
		OriginalStorage: make(map[common.Hash]common.Hash),
		CodeHash:        crypto.Keccak256(nil), // Empty code hash
	}
}

type journalEntry interface {
	revert(*MemoryStateDB)
}

type storageChange struct {
	account common.Address
	key     common.Hash
	pre     common.Hash
}

func (ch storageChange) revert(db *MemoryStateDB) {
	db.accounts[ch.account].Storage[ch.key] = ch.pre
}

type balanceChange struct {
	account common.Address
	pre     *big.Int
}

func (ch balanceChange) revert(db *MemoryStateDB) {
	db.accounts[ch.account].Balance = ch.pre
}

type nonceChange struct {
	account common.Address
	pre     uint64
}

func (ch nonceChange) revert(db *MemoryStateDB) {
	db.accounts[ch.account].Nonce = ch.pre
}

type codeChange struct {
	account common.Address
	preCode []byte
	preHash []byte
}

func (ch codeChange) revert(db *MemoryStateDB) {
	db.accounts[ch.account].Code = ch.preCode
	db.accounts[ch.account].CodeHash = ch.preHash
}

type createAccountChange struct {
	account common.Address
}

func (ch createAccountChange) revert(db *MemoryStateDB) {
	delete(db.accounts, ch.account)
}

type suicideChange struct {
	account     common.Address
	pre         bool
	preBal      *big.Int
	preNonce    uint64
	preCode     []byte
	preCodeHash []byte
	preStorage  map[common.Hash]common.Hash
}

func (ch suicideChange) revert(db *MemoryStateDB) {
	acc := db.accounts[ch.account]
	if acc != nil {
		acc.Suicided = ch.pre
		acc.Balance = ch.preBal
		acc.Nonce = ch.preNonce
		acc.Code = ch.preCode
		acc.CodeHash = ch.preCodeHash
		acc.Storage = ch.preStorage
	}
}

type refundChange struct {
	pre uint64
}

func (ch refundChange) revert(db *MemoryStateDB) {
	db.refund = ch.pre
}

type MemoryStateDB struct {
	accounts map[common.Address]*Account
	journal  []journalEntry
	refund   uint64
	// Access List (EIP-2929)
	accessListAddrs map[common.Address]struct{}
	accessListSlots map[common.Address]map[common.Hash]struct{}

	// Transient Storage (EIP-1153)
	transientStorage map[common.Address]map[common.Hash]common.Hash
}

func NewMemoryStateDB() *MemoryStateDB {
	return &MemoryStateDB{
		accounts:        make(map[common.Address]*Account),
		journal:         make([]journalEntry, 0),
		refund:          0,
		accessListAddrs: make(map[common.Address]struct{}),
		accessListSlots: make(map[common.Address]map[common.Hash]struct{}),
		transientStorage: make(map[common.Address]map[common.Hash]common.Hash),
	}
}

func (db *MemoryStateDB) getAccount(addr common.Address) *Account {
	if acc, ok := db.accounts[addr]; ok {
		return acc
	}
	return nil
}

func (db *MemoryStateDB) getOrNewAccount(addr common.Address) *Account {
	acc := db.getAccount(addr)
	if acc == nil {
		acc = NewAccount()
		db.accounts[addr] = acc
		db.journal = append(db.journal, createAccountChange{account: addr})
	}
	return acc
}

func (db *MemoryStateDB) CreateAccount(addr common.Address) {
	db.getOrNewAccount(addr)
}

func (db *MemoryStateDB) SubBalance(addr common.Address, amount *big.Int) {
	acc := db.getOrNewAccount(addr)
	// Journal pre-balance
	db.journal = append(db.journal, balanceChange{
		account: addr,
		pre:     new(big.Int).Set(acc.Balance),
	})
	acc.Balance.Sub(acc.Balance, amount)
}

func (db *MemoryStateDB) AddBalance(addr common.Address, amount *big.Int) {
	acc := db.getOrNewAccount(addr)
	// Journal pre-balance
	db.journal = append(db.journal, balanceChange{
		account: addr,
		pre:     new(big.Int).Set(acc.Balance),
	})
	acc.Balance.Add(acc.Balance, amount)
}

func (db *MemoryStateDB) GetBalance(addr common.Address) *big.Int {
	acc := db.getAccount(addr)
	if acc == nil {
		return common.Big0
	}
	return acc.Balance
}

func (db *MemoryStateDB) GetNonce(addr common.Address) uint64 {
	acc := db.getAccount(addr)
	if acc == nil {
		return 0
	}
	return acc.Nonce
}

func (db *MemoryStateDB) SetNonce(addr common.Address, nonce uint64) {
	acc := db.getOrNewAccount(addr)
	db.journal = append(db.journal, nonceChange{
		account: addr,
		pre:     acc.Nonce,
	})
	acc.Nonce = nonce
}

func (db *MemoryStateDB) GetCodeHash(addr common.Address) common.Hash {
	acc := db.getAccount(addr)
	if acc == nil || len(acc.CodeHash) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(acc.CodeHash)
}

func (db *MemoryStateDB) GetCode(addr common.Address) []byte {
	acc := db.getAccount(addr)
	if acc == nil {
		return nil
	}
	return acc.Code
}

func (db *MemoryStateDB) SetCode(addr common.Address, code []byte) {
	acc := db.getOrNewAccount(addr)
	db.journal = append(db.journal, codeChange{
		account: addr,
		preCode: acc.Code,
		preHash: acc.CodeHash,
	})
	acc.Code = code
	acc.CodeHash = crypto.Keccak256(code)
}

func (db *MemoryStateDB) GetCodeSize(addr common.Address) int {
	acc := db.getAccount(addr)
	if acc == nil {
		return 0
	}
	return len(acc.Code)
}

func (db *MemoryStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	acc := db.getAccount(addr)
	if acc == nil {
		return common.Hash{}
	}
	return acc.Storage[key]
}

func (db *MemoryStateDB) GetOriginalState(addr common.Address, key common.Hash) common.Hash {
	acc := db.getAccount(addr)
	if acc == nil {
		return common.Hash{}
	}
	return acc.OriginalStorage[key]
}

func (db *MemoryStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	acc := db.getOrNewAccount(addr)
	pre := acc.Storage[key]
	db.journal = append(db.journal, storageChange{
		account: addr,
		key:     key,
		pre:     pre,
	})
	acc.Storage[key] = value
}

// GetTransientState returns the value from transient storage
func (db *MemoryStateDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	if db.transientStorage[addr] == nil {
		return common.Hash{}
	}
	return db.transientStorage[addr][key]
}

// SetTransientState sets the value in transient storage
func (db *MemoryStateDB) SetTransientState(addr common.Address, key common.Hash, value common.Hash) {
	if db.transientStorage[addr] == nil {
		db.transientStorage[addr] = make(map[common.Hash]common.Hash)
	}
	db.transientStorage[addr][key] = value
}

// InitState sets the initial state (both current and original). Used for test setup.
func (db *MemoryStateDB) InitState(addr common.Address, key common.Hash, value common.Hash) {
	acc := db.getOrNewAccount(addr)
	acc.Storage[key] = value
	acc.OriginalStorage[key] = value
}

func (db *MemoryStateDB) Suicide(addr common.Address) bool {
	acc := db.getAccount(addr)
	if acc == nil {
		return false
	}
	db.journal = append(db.journal, suicideChange{
		account:     addr,
		pre:         acc.Suicided,
		preBal:      new(big.Int).Set(acc.Balance),
		preNonce:    acc.Nonce,
		preCode:     acc.Code,
		preCodeHash: acc.CodeHash,
		preStorage:  acc.Storage,
	})
	acc.Suicided = true
	acc.Balance = new(big.Int)
	acc.Nonce = 0
	acc.Code = nil
	acc.CodeHash = nil
	acc.Storage = make(map[common.Hash]common.Hash)
	return true
}

func (db *MemoryStateDB) HasSuicided(addr common.Address) bool {
	acc := db.getAccount(addr)
	if acc == nil {
		return false
	}
	return acc.Suicided
}

func (db *MemoryStateDB) HasBeenCreatedInCurrentTx(addr common.Address) bool {
	for _, entry := range db.journal {
		if ch, ok := entry.(createAccountChange); ok && ch.account == addr {
			return true
		}
	}
	return false
}

func (db *MemoryStateDB) Exist(addr common.Address) bool {
	return db.getAccount(addr) != nil
}

func (db *MemoryStateDB) Empty(addr common.Address) bool {
	acc := db.getAccount(addr)
	return acc == nil || (acc.Nonce == 0 && acc.Balance.Sign() == 0 && len(acc.Code) == 0)
}

func (db *MemoryStateDB) AddRefund(gas uint64) {
	db.journal = append(db.journal, refundChange{pre: db.refund})
	db.refund += gas
}

func (db *MemoryStateDB) SubRefund(gas uint64) {
	db.journal = append(db.journal, refundChange{pre: db.refund})
	if gas > db.refund {
		db.refund = 0
	} else {
		db.refund -= gas
	}
}

func (db *MemoryStateDB) ClearJournal() {
	db.journal = make([]journalEntry, 0)
}

func (db *MemoryStateDB) GetRefund() uint64 {
	return db.refund
}

func (db *MemoryStateDB) Snapshot() int {
	return len(db.journal)
}

func (db *MemoryStateDB) RevertToSnapshot(id int) {
	if id < 0 || id > len(db.journal) {
		return
	}
	for i := len(db.journal) - 1; i >= id; i-- {
		db.journal[i].revert(db)
	}
	db.journal = db.journal[:id]
}

func (db *MemoryStateDB) ForEachStorage(addr common.Address, cb func(key, value common.Hash) bool) {
	acc := db.getAccount(addr)
	if acc == nil {
		return
	}
	for k, v := range acc.Storage {
		if !cb(k, v) {
			return
		}
	}
}

func (db *MemoryStateDB) ForEachAccount(cb func(addr common.Address) bool) {
	for addr := range db.accounts {
		if !cb(addr) {
			return
		}
	}
}
