package vm

import (
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func newTransitionTestState(code []byte) (*core.MemoryStateDB, common.Address, common.Address, *BlockContext) {
	state := core.NewMemoryStateDB()
	sender := common.HexToAddress("0x1000000000000000000000000000000000000001")
	recipient := common.HexToAddress("0x2000000000000000000000000000000000000002")
	coinbase := common.HexToAddress("0x3000000000000000000000000000000000000003")
	state.AddBalance(sender, big.NewInt(1_000_000_000))
	state.SetCode(recipient, code)
	ctx := &BlockContext{BlockNumber: big.NewInt(0), GasLimit: 30_000_000, Coinbase: coinbase, ChainID: big.NewInt(1)}
	return state, sender, recipient, ctx
}

func TestApplyTransactionReturnsExceptionalHalt(t *testing.T) {
	state, sender, recipient, ctx := newTransitionTestState([]byte{core.INVALID})
	tx := types.NewTransaction(0, recipient, big.NewInt(100), 50_000, big.NewInt(1), nil)

	_, gasUsed, reverted, err := ApplyTransactionWithContext(state, tx, sender, ctx)

	if err == nil || err.Error() != "invalid opcode: 0xfe" {
		t.Fatalf("error = %v, want invalid opcode", err)
	}
	if !reverted {
		t.Fatal("exceptional halt should mark the transaction reverted")
	}
	if gasUsed != tx.Gas() {
		t.Fatalf("gas used = %d, want %d", gasUsed, tx.Gas())
	}
	if state.GetNonce(sender) != 1 {
		t.Fatalf("sender nonce = %d, want 1", state.GetNonce(sender))
	}
	if state.GetBalance(recipient).Sign() != 0 {
		t.Fatalf("recipient retained reverted value: %s", state.GetBalance(recipient))
	}
}

func TestApplyTransactionKeepsRevertDistinctFromError(t *testing.T) {
	code := []byte{core.PUSH1, 0x00, core.PUSH1, 0x00, core.REVERT}
	state, sender, recipient, ctx := newTransitionTestState(code)
	tx := types.NewTransaction(0, recipient, big.NewInt(100), 50_000, big.NewInt(1), nil)

	_, gasUsed, reverted, err := ApplyTransactionWithContext(state, tx, sender, ctx)

	if err != nil {
		t.Fatalf("REVERT returned execution error: %v", err)
	}
	if !reverted {
		t.Fatal("expected REVERT result")
	}
	if gasUsed != 21_006 {
		t.Fatalf("gas used = %d, want 21006", gasUsed)
	}
	if state.GetBalance(recipient).Sign() != 0 {
		t.Fatalf("recipient retained reverted value: %s", state.GetBalance(recipient))
	}
}

func TestApplyTransactionReturnsOutOfGas(t *testing.T) {
	state, sender, recipient, ctx := newTransitionTestState([]byte{core.PUSH1, 0x01})
	tx := types.NewTransaction(0, recipient, big.NewInt(0), 21_002, big.NewInt(1), nil)

	_, gasUsed, reverted, err := ApplyTransactionWithContext(state, tx, sender, ctx)

	if err == nil || err.Error() != "out of gas: have 2, want 3" {
		t.Fatalf("error = %v, want out of gas", err)
	}
	if !reverted || gasUsed != tx.Gas() {
		t.Fatalf("reverted=%v gasUsed=%d, want true/%d", reverted, gasUsed, tx.Gas())
	}
}

func TestApplyTransactionRunsTopLevelPrecompile(t *testing.T) {
	state, sender, _, ctx := newTransitionTestState(nil)
	to := PrecompileSHA256
	input := []byte("abc")
	tx := types.NewTransaction(0, to, big.NewInt(0), 50_000, big.NewInt(1), input)

	output, gasUsed, reverted, err := ApplyTransactionWithContext(state, tx, sender, ctx)

	if err != nil || reverted {
		t.Fatalf("precompile failed: reverted=%v err=%v", reverted, err)
	}
	want := sha256.Sum256(input)
	if common.BytesToHash(output) != common.BytesToHash(want[:]) {
		t.Fatalf("output = %x, want %x", output, want)
	}
	if gasUsed != 21_120 {
		t.Fatalf("gas used = %d, want 21120", gasUsed)
	}
}

func TestApplyTransactionRejectsInsufficientValueWithoutMutation(t *testing.T) {
	state, sender, recipient, ctx := newTransitionTestState(nil)
	state.SubBalance(sender, new(big.Int).Sub(state.GetBalance(sender), big.NewInt(50_000)))
	tx := types.NewTransaction(0, recipient, big.NewInt(1), 50_000, big.NewInt(1), nil)
	before := new(big.Int).Set(state.GetBalance(sender))

	_, _, _, err := ApplyTransactionWithContext(state, tx, sender, ctx)

	if err == nil {
		t.Fatal("expected insufficient funds error")
	}
	if state.GetBalance(sender).Cmp(before) != 0 || state.GetNonce(sender) != 0 {
		t.Fatalf("pre-check mutated sender: balance=%s nonce=%d", state.GetBalance(sender), state.GetNonce(sender))
	}
}

func TestApplyTransactionRejectsIntrinsicGasWithoutMutation(t *testing.T) {
	state, sender, recipient, ctx := newTransitionTestState(nil)
	tx := types.NewTransaction(0, recipient, big.NewInt(0), 20_999, big.NewInt(1), nil)
	before := new(big.Int).Set(state.GetBalance(sender))

	_, _, _, err := ApplyTransactionWithContext(state, tx, sender, ctx)

	if err == nil {
		t.Fatal("expected intrinsic gas error")
	}
	if state.GetBalance(sender).Cmp(before) != 0 || state.GetNonce(sender) != 0 {
		t.Fatalf("pre-check mutated sender: balance=%s nonce=%d", state.GetBalance(sender), state.GetNonce(sender))
	}
}

func TestApplyTransactionHandlesUint64GasWithoutSignedOverflow(t *testing.T) {
	state, sender, recipient, ctx := newTransitionTestState(nil)
	gasLimit := ^uint64(0)
	initialBalance := new(big.Int).SetUint64(gasLimit)
	state.AddBalance(sender, new(big.Int).Sub(initialBalance, state.GetBalance(sender)))
	tx := types.NewTransaction(0, recipient, big.NewInt(0), gasLimit, big.NewInt(1), nil)

	_, gasUsed, reverted, err := ApplyTransactionWithContext(state, tx, sender, ctx)

	if err != nil || reverted {
		t.Fatalf("transaction failed: reverted=%v err=%v", reverted, err)
	}
	if gasUsed != 21_000 {
		t.Fatalf("gas used = %d, want 21000", gasUsed)
	}
	wantBalance := new(big.Int).Sub(initialBalance, new(big.Int).SetUint64(gasUsed))
	if state.GetBalance(sender).Cmp(wantBalance) != 0 {
		t.Fatalf("sender balance = %s, want %s", state.GetBalance(sender), wantBalance)
	}
}
