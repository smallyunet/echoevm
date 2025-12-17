package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// StateDB defines the interface for accessing and modifying the state.
// It is a simplified version of the go-ethereum StateDB interface.
type StateDB interface {
	CreateAccount(addr common.Address)

	SubBalance(addr common.Address, amount *big.Int)
	AddBalance(addr common.Address, amount *big.Int)
	GetBalance(addr common.Address) *big.Int

	GetNonce(addr common.Address) uint64
	SetNonce(addr common.Address, nonce uint64)

	GetCodeHash(addr common.Address) common.Hash
	GetCode(addr common.Address) []byte
	SetCode(addr common.Address, code []byte)
	GetCodeSize(addr common.Address) int

	// Storage
	GetState(addr common.Address, key common.Hash) common.Hash
	GetOriginalState(addr common.Address, key common.Hash) common.Hash
	SetState(addr common.Address, key common.Hash, value common.Hash)

	// Transient Storage (EIP-1153)
	GetTransientState(addr common.Address, key common.Hash) common.Hash
	SetTransientState(addr common.Address, key common.Hash, value common.Hash)

	// Suicide (Selfdestruct)
	Suicide(addr common.Address) bool
	HasSuicided(addr common.Address) bool
	// HasBeenCreatedInCurrentTx checks if the account was created in the current transaction (EIP-6780)
	HasBeenCreatedInCurrentTx(addr common.Address) bool

	// Existence
	Exist(addr common.Address) bool
	Empty(addr common.Address) bool

	// Refund
	AddRefund(gas uint64)
	SubRefund(gas uint64)
	GetRefund() uint64

	// Access List
	AddAddressToAccessList(addr common.Address)
	AddSlotToAccessList(addr common.Address, slot common.Hash)
	AddressInAccessList(addr common.Address) bool
	SlotInAccessList(addr common.Address, slot common.Hash) bool

	// Snapshot/Revert (optional for now, but good to have in interface)
	Snapshot() int
	RevertToSnapshot(int)
}
