package core

import (
	"errors"
	"math/big"
	"testing"

	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/crypto"
)

type failingStateBackend struct{ err error }

func (b failingStateBackend) GetAccount(common.Address) (*Account, error) { return nil, b.err }
func (b failingStateBackend) GetStorage(common.Address, common.Hash) (common.Hash, error) {
	return common.Hash{}, b.err
}

func TestMemoryStateDB(t *testing.T) {
	db := NewMemoryStateDB()
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	// Test Balance
	db.AddBalance(addr, big.NewInt(100))
	if db.GetBalance(addr).Cmp(big.NewInt(100)) != 0 {
		t.Errorf("expected balance 100, got %v", db.GetBalance(addr))
	}
	db.SubBalance(addr, big.NewInt(50))
	if db.GetBalance(addr).Cmp(big.NewInt(50)) != 0 {
		t.Errorf("expected balance 50, got %v", db.GetBalance(addr))
	}

	// Test Nonce
	db.SetNonce(addr, 42)
	if db.GetNonce(addr) != 42 {
		t.Errorf("expected nonce 42, got %v", db.GetNonce(addr))
	}

	// Test Code
	code := []byte{0x01, 0x02, 0x03}
	db.SetCode(addr, code)
	if len(db.GetCode(addr)) != 3 {
		t.Errorf("expected code length 3, got %v", len(db.GetCode(addr)))
	}
	expectedHash := crypto.Keccak256(code)
	if db.GetCodeHash(addr) != common.BytesToHash(expectedHash) {
		t.Errorf("code hash mismatch")
	}

	// Test Storage
	key := common.HexToHash("0x01")
	val := common.HexToHash("0x02")
	db.SetState(addr, key, val)
	if db.GetState(addr, key) != val {
		t.Errorf("expected storage value %v, got %v", val, db.GetState(addr, key))
	}
}

func TestMemoryStateDBRecordsBackendFailures(t *testing.T) {
	db := NewMemoryStateDB()
	want := errors.New("proof unavailable")
	db.SetBackend(failingStateBackend{err: want})
	_ = db.GetBalance(common.HexToAddress("0x1234"))
	if !errors.Is(db.BackendError(), want) {
		t.Fatalf("backend error = %v, want %v", db.BackendError(), want)
	}
}

func TestTransientStorageRevertsWithSnapshot(t *testing.T) {
	db := NewMemoryStateDB()
	addr := common.HexToAddress("0x1000")
	key := common.HexToHash("0x01")
	initial := common.HexToHash("0x02")
	db.SetTransientState(addr, key, initial)
	snapshot := db.Snapshot()
	db.SetTransientState(addr, key, common.HexToHash("0x03"))
	db.SetTransientState(addr, common.HexToHash("0x04"), common.HexToHash("0x05"))
	db.RevertToSnapshot(snapshot)

	if got := db.GetTransientState(addr, key); got != initial {
		t.Fatalf("transient value = %s, want %s", got, initial)
	}
	if got := db.GetTransientState(addr, common.HexToHash("0x04")); got != (common.Hash{}) {
		t.Fatalf("new transient slot survived revert: %s", got)
	}
}

func TestPrepareTransactionResetsTransactionScopedState(t *testing.T) {
	db := NewMemoryStateDB()
	addr := common.HexToAddress("0x1234")
	key := common.HexToHash("0x01")
	value := common.HexToHash("0x02")

	db.SetState(addr, key, value)
	db.SetTransientState(addr, key, value)
	db.AddAddressToAccessList(addr)
	db.AddSlotToAccessList(addr, key)
	db.AddRefund(100)

	db.PrepareTransaction()

	if db.GetRefund() != 0 {
		t.Fatalf("refund was not reset: %d", db.GetRefund())
	}
	if db.AddressInAccessList(addr) || db.SlotInAccessList(addr, key) {
		t.Fatal("access list was not reset")
	}
	if got := db.GetTransientState(addr, key); got != (common.Hash{}) {
		t.Fatalf("transient storage was not reset: %s", got.Hex())
	}
	if got := db.GetOriginalState(addr, key); got != value {
		t.Fatalf("original storage = %s, want %s", got.Hex(), value.Hex())
	}
	if db.HasBeenCreatedInCurrentTx(addr) {
		t.Fatal("journal entries leaked into the new transaction")
	}
}

func TestCreatedInCurrentTransactionIsExplicitAndRevertible(t *testing.T) {
	db := NewMemoryStateDB()
	addr := common.HexToAddress("0x1234")
	db.PrepareTransaction()
	db.CreateAccount(addr)
	if db.HasBeenCreatedInCurrentTx(addr) {
		t.Fatal("ordinary account materialization must not trigger EIP-6780 deletion semantics")
	}
	snapshot := db.Snapshot()
	db.MarkCreatedInCurrentTx(addr)
	if !db.HasBeenCreatedInCurrentTx(addr) {
		t.Fatal("explicit CREATE marker was not recorded")
	}
	db.RevertToSnapshot(snapshot)
	if db.HasBeenCreatedInCurrentTx(addr) {
		t.Fatal("CREATE marker survived snapshot revert")
	}
}

func TestSelfDestructDeletionIsDeferredUntilTransactionFinalization(t *testing.T) {
	db := NewMemoryStateDB()
	addr := common.HexToAddress("0x1234")
	code := []byte{0x60, 0x00}
	db.SetNonce(addr, 1)
	db.SetCode(addr, code)
	db.PrepareTransaction()
	db.MarkCreatedInCurrentTx(addr)
	db.Suicide(addr)
	if got := db.GetCode(addr); len(got) != len(code) {
		t.Fatalf("code was cleared during execution: %x", got)
	}
	db.FinalizeTransaction()
	if db.Exist(addr) {
		t.Fatal("self-destructed account survived transaction finalization")
	}
}
