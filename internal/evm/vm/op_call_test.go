package vm

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smallyunet/echoevm/internal/evm/core"
)

func TestOpDelegateCall(t *testing.T) {
	i := newInterp()
	// Push in EVM stack order so the target is an empty non-precompile account.
	i.stack.PushSafe(new(big.Int)) // retLength
	i.stack.PushSafe(new(big.Int)) // retOffset
	i.stack.PushSafe(new(big.Int)) // argsLength
	i.stack.PushSafe(new(big.Int)) // argsOffset
	i.stack.PushSafe(big.NewInt(0x20))
	i.stack.PushSafe(big.NewInt(100_000))
	opDelegateCall(i, 0)
	// With no code to execute, delegatecall should succeed (push 1)
	result := i.stack.PopSafe()
	if result.Sign() != 1 {
		t.Fatalf("delegatecall with empty code should succeed, got %v", result)
	}
}

func TestStaticCallRejectsStateWrite(t *testing.T) {
	state := core.NewMemoryStateDB()
	caller := common.HexToAddress("0x1000")
	target := common.HexToAddress("0x2000")
	key := common.Hash{}
	state.SetCode(target, []byte{core.PUSH1, 0x01, core.PUSH0, core.SSTORE, core.STOP})
	state.PrepareTransaction()
	state.AddAddressToAccessList(target)

	i := New(nil, state, caller)
	i.SetGas(100_000)
	pushStaticCallArguments(i, target, 50_000)
	opStaticCall(i, core.STATICCALL)

	if got := i.stack.PopSafe(); got.Sign() != 0 {
		t.Fatalf("STATICCALL result = %s, want failure", got)
	}
	if got := state.GetState(target, key); got != (common.Hash{}) {
		t.Fatalf("STATICCALL changed storage: %s", got)
	}
	if i.Err() != nil {
		t.Fatalf("child write protection escaped into parent: %v", i.Err())
	}
}

func TestReadOnlyFrameRejectsMutatingOpcodes(t *testing.T) {
	tests := []struct {
		name string
		op   OpcodeHandler
		code byte
	}{
		{name: "SSTORE", op: opSstore, code: core.SSTORE},
		{name: "TSTORE", op: opTstore, code: core.TSTORE},
		{name: "LOG0", op: opLog, code: core.LOG0},
		{name: "CREATE", op: opCreate, code: core.CREATE},
		{name: "CREATE2", op: opCreate2, code: core.CREATE2},
		{name: "SELFDESTRUCT", op: opSelfDestruct, code: core.SELFDESTRUCT},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := newInterp()
			i.SetReadOnly(true)
			tt.op(i, tt.code)
			if !errors.Is(i.Err(), ErrWriteProtection) {
				t.Fatalf("error = %v, want write protection", i.Err())
			}
			if !i.IsReverted() {
				t.Fatal("write protection must exceptionally halt the frame")
			}
		})
	}
}

func TestReadOnlyCallRejectsValueTransfer(t *testing.T) {
	i := newInterp()
	i.SetReadOnly(true)
	// retLength, retOffset, argsLength, argsOffset, value, address, gas
	for _, value := range []*big.Int{
		big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(1), big.NewInt(2), big.NewInt(50_000),
	} {
		i.stack.PushSafe(value)
	}
	opCall(i, core.CALL)
	if !errors.Is(i.Err(), ErrWriteProtection) {
		t.Fatalf("error = %v, want write protection", i.Err())
	}
}

func TestCreateSuccessRefundsGasAndStoresRuntimeCode(t *testing.T) {
	creator := common.HexToAddress("0x1000")
	state := core.NewMemoryStateDB()
	state.AddBalance(creator, big.NewInt(10))
	state.PrepareTransaction()

	// Store one zero byte in memory and return it as runtime code.
	initCode := []byte{core.PUSH1, 0x00, core.PUSH1, 0x00, core.MSTORE8, core.PUSH1, 0x01, core.PUSH1, 0x00, core.RETURN}
	i := New(nil, state, creator)
	i.SetGas(100_000)
	i.memory.Write(0, initCode)
	pushCreateArguments(i, big.NewInt(3), uint64(len(initCode)))
	opCreate(i, core.CREATE)

	addr := crypto.CreateAddress(creator, 0)
	if got := common.BigToAddress(i.stack.PopSafe()); got != addr {
		t.Fatalf("created address = %s, want %s", got, addr)
	}
	if got := state.GetCode(addr); len(got) != 1 || got[0] != 0x00 {
		t.Fatalf("runtime code = %x, want 00", got)
	}
	if got := state.GetBalance(addr); got.Cmp(big.NewInt(3)) != 0 {
		t.Fatalf("created balance = %s, want 3", got)
	}
	if got := state.GetNonce(creator); got != 1 {
		t.Fatalf("creator nonce = %d, want 1", got)
	}
	if i.Gas() <= 100_000/64 || i.Gas() >= 100_000 {
		t.Fatalf("remaining gas = %d, want refunded child gas minus execution/deposit cost", i.Gas())
	}
}

func TestCreateFailuresRestoreState(t *testing.T) {
	tests := []struct {
		name       string
		initCode   []byte
		wantRefund bool
	}{
		{
			name:       "revert preserves child gas",
			initCode:   []byte{core.PUSH1, 0x01, core.PUSH0, core.SSTORE, core.PUSH0, core.PUSH0, core.REVERT},
			wantRefund: true,
		},
		{
			name:     "exceptional halt burns child gas",
			initCode: []byte{core.PUSH1, 0x01, core.PUSH0, core.SSTORE, core.INVALID},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := common.HexToAddress("0x1000")
			state := core.NewMemoryStateDB()
			state.AddBalance(creator, big.NewInt(10))
			state.PrepareTransaction()
			i := New(nil, state, creator)
			i.SetGas(100_000)
			i.memory.Write(0, tt.initCode)
			pushCreateArguments(i, big.NewInt(3), uint64(len(tt.initCode)))
			opCreate(i, core.CREATE)

			addr := crypto.CreateAddress(creator, 0)
			if got := i.stack.PopSafe(); got.Sign() != 0 {
				t.Fatalf("CREATE result = %s, want failure", got)
			}
			if state.Exist(addr) {
				t.Fatalf("failed CREATE leaked account %s", addr)
			}
			if got := state.GetBalance(creator); got.Cmp(big.NewInt(10)) != 0 {
				t.Fatalf("creator balance = %s, want restored 10", got)
			}
			if got := state.GetNonce(creator); got != 1 {
				t.Fatalf("creator nonce = %d, want failed attempt to consume nonce", got)
			}
			reserve := uint64(100_000) / 64
			if tt.wantRefund && i.Gas() <= reserve {
				t.Fatalf("REVERT remaining gas = %d, want child refund above reserve %d", i.Gas(), reserve)
			}
			if !tt.wantRefund && i.Gas() > reserve {
				t.Fatalf("exceptional halt remaining gas = %d, want forwarded gas burned", i.Gas())
			}
		})
	}
}

func TestCreate2FailureRestoresState(t *testing.T) {
	creator := common.HexToAddress("0x1000")
	state := core.NewMemoryStateDB()
	state.AddBalance(creator, big.NewInt(10))
	state.PrepareTransaction()
	initCode := []byte{core.PUSH0, core.PUSH0, core.REVERT}
	salt := big.NewInt(7)
	i := New(nil, state, creator)
	i.SetGas(100_000)
	i.memory.Write(0, initCode)
	// salt, length, offset, value are popped in reverse push order.
	i.stack.PushSafe(salt)
	i.stack.PushSafe(new(big.Int).SetUint64(uint64(len(initCode))))
	i.stack.PushSafe(big.NewInt(0))
	i.stack.PushSafe(big.NewInt(3))
	opCreate2(i, core.CREATE2)

	var saltBytes [32]byte
	salt.FillBytes(saltBytes[:])
	addr := crypto.CreateAddress2(creator, saltBytes, crypto.Keccak256(initCode))
	if got := i.stack.PopSafe(); got.Sign() != 0 {
		t.Fatalf("CREATE2 result = %s, want failure", got)
	}
	if state.Exist(addr) {
		t.Fatalf("failed CREATE2 leaked account %s", addr)
	}
	if got := state.GetNonce(creator); got != 1 {
		t.Fatalf("creator nonce = %d, want 1", got)
	}
}

func pushStaticCallArguments(i *Interpreter, target common.Address, gas uint64) {
	for _, value := range []*big.Int{
		big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), target.Big(), new(big.Int).SetUint64(gas),
	} {
		i.stack.PushSafe(value)
	}
}

func pushCreateArguments(i *Interpreter, value *big.Int, length uint64) {
	i.stack.PushSafe(new(big.Int).SetUint64(length))
	i.stack.PushSafe(big.NewInt(0))
	i.stack.PushSafe(value)
}
