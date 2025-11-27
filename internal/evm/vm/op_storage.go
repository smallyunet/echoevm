package vm

import (
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func opSload(i *Interpreter, _ byte) {
	key := i.stack.PopSafe()
	// TODO: Address is currently hardcoded or missing context.
	// For now, we assume a single context or need to add address to Interpreter.
	// Since we don't have "current address" in Interpreter yet, we might need to add it.
	// But wait, the previous implementation was just a map.
	// Let's use a dummy address for now or add Address to Interpreter.
	// Adding Address to Interpreter is better.

	// For this step, I will assume we need to add Address to Interpreter.
	// But to keep it simple and compile, I'll use a zero address if I haven't added it yet.
	// Actually, I should add Address to Interpreter.

	// Let's check if I can add Address to Interpreter in the previous step.
	// I didn't. So I should add it now or in a separate step.
	// I'll use a placeholder address for now to make it compile, then fix it.
	// Actually, the previous implementation was `storage map[string]*big.Int`.
	// It was effectively a single account storage.
	// So I should use a specific address for the "current execution context".

	val := i.statedb.GetState(i.address, common.BigToHash(key))
	i.stack.PushSafe(val.Big())
}

func opSstore(i *Interpreter, _ byte) {
	key := i.stack.PopSafe()
	val := i.stack.PopSafe()
	i.statedb.SetState(i.address, common.BigToHash(key), common.BigToHash(val))
}

func storageKey(k *big.Int) string {
	b := make([]byte, 32)
	k.FillBytes(b)
	return hex.EncodeToString(b)
}
