package types

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/crypto"
	"github.com/smallyunet/echoevm/internal/eth/rlp"
)

const delegationPrefix = "\xef\x01\x00"

type SetCodeAuthorization struct {
	ChainID uint256.Int
	Address common.Address
	Nonce   uint64
	V       uint8
	R, S    uint256.Int
}

func AddressToDelegation(address common.Address) []byte {
	return append([]byte(delegationPrefix), address[:]...)
}
func ParseDelegation(code []byte) (common.Address, bool) {
	if len(code) != 23 || string(code[:3]) != delegationPrefix {
		return common.Address{}, false
	}
	return common.BytesToAddress(code[3:]), true
}
func SignSetCode(key *ecdsa.PrivateKey, auth SetCodeAuthorization) (SetCodeAuthorization, error) {
	hash := authorizationHash(auth)
	sig, err := crypto.Sign(hash[:], key)
	if err != nil {
		return auth, err
	}
	auth.V = sig[64]
	auth.R.SetBytes(sig[:32])
	auth.S.SetBytes(sig[32:64])
	return auth, nil
}
func (a *SetCodeAuthorization) Authority() (common.Address, error) {
	hash := authorizationHash(*a)
	sig := make([]byte, 65)
	a.R.WriteToSlice(sig[:32])
	a.S.WriteToSlice(sig[32:64])
	sig[64] = a.V
	pub, err := crypto.SigToPub(hash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pub), nil
}
func authorizationHash(a SetCodeAuthorization) common.Hash {
	chain := a.ChainID.ToBig()
	enc := rlp.EncodeList(rlp.EncodeBytes(chain.Bytes()), rlp.EncodeBytes(a.Address[:]), rlp.EncodeBytes(uintBytes(a.Nonce)))
	return crypto.Keccak256Hash([]byte{0x05}, enc)
}

func encodeAuthorizations(list []SetCodeAuthorization) []byte {
	out := make([][]byte, len(list))
	for i, a := range list {
		out[i] = rlp.EncodeList(rlp.EncodeBytes(a.ChainID.Bytes()), rlp.EncodeBytes(a.Address[:]), rlp.EncodeBytes(uintBytes(a.Nonce)), rlp.EncodeBytes(uintBytes(uint64(a.V))), rlp.EncodeBytes(a.R.Bytes()), rlp.EncodeBytes(a.S.Bytes()))
	}
	return rlp.EncodeList(out...)
}
func decodeAuthorizations(raw rlp.RawValue) ([]SetCodeAuthorization, error) {
	entries, err := rlp.SplitList(raw)
	if err != nil {
		return nil, err
	}
	out := make([]SetCodeAuthorization, len(entries))
	for i, e := range entries {
		f, err := rlp.SplitList(e)
		if err != nil || len(f) != 6 {
			return nil, fmt.Errorf("invalid authorization")
		}
		get := func(j int) []byte { b, _ := rlp.Bytes(f[j]); return b }
		out[i].ChainID.SetBytes(get(0))
		out[i].Address = common.BytesToAddress(get(1))
		out[i].Nonce = new(big.Int).SetBytes(get(2)).Uint64()
		out[i].V = uint8(new(big.Int).SetBytes(get(3)).Uint64())
		out[i].R.SetBytes(get(4))
		out[i].S.SetBytes(get(5))
	}
	return out, nil
}
