package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func opTstore(i *Interpreter, op byte) {
	key, err := i.stack.Pop()
	if err != nil {
		i.err = err
		return
	}
	val, err := i.stack.Pop()
	if err != nil {
		i.err = err
		return
	}

	i.statedb.SetTransientState(i.address, common.BytesToHash(key.Bytes()), common.BytesToHash(val.Bytes()))
}

func opTload(i *Interpreter, op byte) {
	key, err := i.stack.Pop()
	if err != nil {
		i.err = err
		return
	}
	val := i.statedb.GetTransientState(i.address, common.BytesToHash(key.Bytes()))
	
	// Convert [32]byte to big.Int and push
	i.stack.Push(new(big.Int).SetBytes(val.Bytes()))
}
