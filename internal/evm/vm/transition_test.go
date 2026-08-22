package vm

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/crypto"
	"github.com/smallyunet/echoevm/internal/eth/params"
	"github.com/smallyunet/echoevm/internal/eth/types"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func mustTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	return key
}

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

func TestApplyTransactionUsesDynamicFeeEffectiveGasPrice(t *testing.T) {
	state, sender, recipient, ctx := newTransitionTestState(nil)
	ctx.BaseFee = big.NewInt(2)
	initial := new(big.Int).Set(state.GetBalance(sender))
	tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(1), Nonce: 0, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(10), Gas: 21_000, To: &recipient})

	_, gasUsed, reverted, err := ApplyTransactionWithContext(state, tx, sender, ctx)
	if err != nil || reverted || gasUsed != 21_000 {
		t.Fatalf("reverted=%v gas=%d err=%v", reverted, gasUsed, err)
	}
	wantSender := new(big.Int).Sub(initial, big.NewInt(63_000))
	if state.GetBalance(sender).Cmp(wantSender) != 0 {
		t.Fatalf("sender balance=%s want=%s", state.GetBalance(sender), wantSender)
	}
	if state.GetBalance(ctx.Coinbase).Cmp(big.NewInt(21_000)) != 0 {
		t.Fatalf("coinbase balance=%s want=21000", state.GetBalance(ctx.Coinbase))
	}
}

func TestApplyTransactionRejectsFeeCapBelowBaseFeeWithoutMutation(t *testing.T) {
	state, sender, recipient, ctx := newTransitionTestState(nil)
	ctx.BaseFee = big.NewInt(11)
	tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(1), GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(10), Gas: 21_000, To: &recipient})
	before := new(big.Int).Set(state.GetBalance(sender))

	if _, _, _, err := ApplyTransactionWithContext(state, tx, sender, ctx); err == nil {
		t.Fatal("expected fee-cap rejection")
	}
	if state.GetNonce(sender) != 0 || state.GetBalance(sender).Cmp(before) != 0 {
		t.Fatal("fee validation mutated sender state")
	}
}

func TestApplyTransactionChargesInitCodeWordGas(t *testing.T) {
	state, sender, _, ctx := newTransitionTestState(nil)
	tx := types.NewContractCreation(0, big.NewInt(0), 100_000, big.NewInt(1), []byte{core.STOP})

	_, gasUsed, reverted, err := ApplyTransactionWithContext(state, tx, sender, ctx)
	if err != nil || reverted {
		t.Fatalf("reverted=%v err=%v", reverted, err)
	}
	if gasUsed != 53_006 {
		t.Fatalf("gas used=%d want=53006", gasUsed)
	}
}

func TestApplyTransactionRejectsForbiddenRuntimeCodePrefix(t *testing.T) {
	state, sender, _, ctx := newTransitionTestState(nil)
	initCode := []byte{core.PUSH1, 0xef, core.PUSH0, core.MSTORE, core.PUSH1, 0x01, core.PUSH1, 0x1f, core.RETURN}
	tx := types.NewContractCreation(0, big.NewInt(0), 100_000, big.NewInt(1), initCode)

	_, gasUsed, reverted, err := ApplyTransactionWithContext(state, tx, sender, ctx)
	if err == nil || !reverted {
		t.Fatalf("reverted=%v err=%v, want forbidden-prefix rejection", reverted, err)
	}
	if gasUsed != tx.Gas() {
		t.Fatalf("gas used=%d want=%d", gasUsed, tx.Gas())
	}
}

func TestApplyPragueSetCodeTransaction(t *testing.T) {
	senderKey := mustTestKey(t)
	authorityKey := mustTestKey(t)
	sender := crypto.PubkeyToAddress(senderKey.PublicKey)
	authority := crypto.PubkeyToAddress(authorityKey.PublicKey)
	target := common.HexToAddress("0x1000000000000000000000000000000000000001")

	state := core.NewMemoryStateDB()
	state.CreateAccount(sender)
	state.AddBalance(sender, big.NewInt(1_000_000))
	state.SetCode(target, []byte{core.PUSH1, 0x2a, core.PUSH0, core.MSTORE, core.PUSH1, 0x20, core.PUSH0, core.RETURN})

	auth, err := types.SignSetCode(authorityKey, types.SetCodeAuthorization{ChainID: *uint256.NewInt(1), Address: target})
	if err != nil {
		t.Fatal(err)
	}
	tx := types.NewTx(&types.SetCodeTx{
		ChainID:   uint256.NewInt(1),
		GasTipCap: uint256.NewInt(1),
		GasFeeCap: uint256.NewInt(1),
		Gas:       200_000,
		To:        authority,
		Value:     uint256.NewInt(0),
		AuthList:  []types.SetCodeAuthorization{auth},
	})
	config, _ := core.ChainConfigForFork(core.ForkPrague)
	ctx := &BlockContext{BlockNumber: new(big.Int), GasLimit: 30_000_000, ChainID: big.NewInt(1), ChainConfig: config}

	ret, _, reverted, err := ApplyTransactionWithContext(state, tx, sender, ctx)
	if err != nil || reverted {
		t.Fatalf("Prague set-code execution failed: reverted=%v err=%v", reverted, err)
	}
	if len(ret) != 32 || ret[31] != 0x2a {
		t.Fatalf("delegated return data = %x", ret)
	}
	if delegated, ok := types.ParseDelegation(state.GetCode(authority)); !ok || delegated != target {
		t.Fatalf("authority delegation = %s, active=%v", delegated, ok)
	}
	if state.GetNonce(authority) != 1 {
		t.Fatalf("authority nonce = %d, want 1", state.GetNonce(authority))
	}
}

func TestApplyOsakaRejectsTransactionAboveGasCap(t *testing.T) {
	sender := common.HexToAddress("0x100")
	recipient := common.HexToAddress("0x200")
	state := core.NewMemoryStateDB()
	state.CreateAccount(sender)
	state.AddBalance(sender, new(big.Int).Lsh(big.NewInt(1), 100))
	tx := types.NewTransaction(0, recipient, new(big.Int), params.MaxTxGas+1, big.NewInt(1), nil)
	config, _ := core.ChainConfigForFork(core.ForkOsaka)
	ctx := &BlockContext{BlockNumber: new(big.Int), GasLimit: 60_000_000, ChainID: big.NewInt(1), ChainConfig: config}
	if _, _, _, err := ApplyTransactionWithContext(state, tx, sender, ctx); err == nil {
		t.Fatal("expected Osaka transaction gas-cap rejection")
	}
}

func TestApplyOsakaRejectsTooManyBlobs(t *testing.T) {
	sender := common.HexToAddress("0x100")
	recipient := common.HexToAddress("0x200")
	state := core.NewMemoryStateDB()
	state.CreateAccount(sender)
	state.AddBalance(sender, new(big.Int).Lsh(big.NewInt(1), 100))
	hashes := make([]common.Hash, params.BlobTxMaxBlobs+1)
	for index := range hashes {
		hashes[index][0] = 0x01
	}
	tx := types.NewTx(&types.BlobTx{
		ChainID: uint256.NewInt(1), GasTipCap: uint256.NewInt(1), GasFeeCap: uint256.NewInt(1),
		Gas: 100_000, To: recipient, Value: uint256.NewInt(0), BlobFeeCap: uint256.NewInt(1), BlobHashes: hashes,
	})
	config, _ := core.ChainConfigForFork(core.ForkOsaka)
	ctx := &BlockContext{BlockNumber: new(big.Int), GasLimit: 60_000_000, ChainID: big.NewInt(1), ChainConfig: config, BlobBaseFee: big.NewInt(1)}
	if _, _, _, err := ApplyTransactionWithContext(state, tx, sender, ctx); err == nil {
		t.Fatal("expected Osaka blob-count rejection")
	}
}
