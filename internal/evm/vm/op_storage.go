package vm

import (
	"encoding/hex"
	"math/big"
)

func opSload(i *Interpreter, _ byte) {
	key := i.stack.PopSafe()
	k := storageKey(key)
	if val, ok := i.storage[k]; ok {
		i.stack.PushSafe(new(big.Int).Set(val))
	} else {
		i.stack.PushSafe(big.NewInt(0))
	}
}

func opSstore(i *Interpreter, _ byte) {
	val := i.stack.PopSafe()
	key := i.stack.PopSafe()
	k := storageKey(key)
	i.storage[k] = new(big.Int).Set(val)
}

func storageKey(k *big.Int) string {
	b := make([]byte, 32)
	k.FillBytes(b)
	return hex.EncodeToString(b)
}
