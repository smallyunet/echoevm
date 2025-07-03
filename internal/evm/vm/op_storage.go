package vm

import (
	"encoding/hex"
	"math/big"
)

func opSload(i *Interpreter, _ byte) {
	key := i.stack.Pop()
	k := storageKey(key)
	if val, ok := i.storage[k]; ok {
		i.stack.Push(new(big.Int).Set(val))
	} else {
		i.stack.Push(big.NewInt(0))
	}
}

func opSstore(i *Interpreter, _ byte) {
	val := i.stack.Pop()
	key := i.stack.Pop()
	k := storageKey(key)
	i.storage[k] = new(big.Int).Set(val)
}

func storageKey(k *big.Int) string {
	b := make([]byte, 32)
	k.FillBytes(b)
	return hex.EncodeToString(b)
}
