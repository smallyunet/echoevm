package vm

import (
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func opSload(i *Interpreter, _ byte) {
	key := i.stack.PopSafe()
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
