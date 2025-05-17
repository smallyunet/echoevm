package common

import "math/big"

type (
	Address [20]byte
	Hash    [32]byte
	U256    = *big.Int
)

func NewU256(v uint64) U256 { return big.NewInt(0).SetUint64(v) }
