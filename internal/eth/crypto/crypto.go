package crypto

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/smallyunet/echoevm/internal/eth/common"
	"golang.org/x/crypto/sha3"
)

func Keccak256(parts ...[]byte) []byte {
	h := sha3.NewLegacyKeccak256()
	for _, p := range parts {
		_, _ = h.Write(p)
	}
	return h.Sum(nil)
}
func Keccak256Hash(parts ...[]byte) common.Hash { return common.BytesToHash(Keccak256(parts...)) }

func GenerateKey() (*ecdsa.PrivateKey, error) {
	key, err := secp.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	return key.ToECDSA(), nil
}
func ToECDSA(data []byte) (*ecdsa.PrivateKey, error) {
	if len(data) != 32 {
		return nil, fmt.Errorf("invalid private key length %d", len(data))
	}
	d := new(big.Int).SetBytes(data)
	if d.Sign() <= 0 || d.Cmp(secp.Params().N) >= 0 {
		return nil, fmt.Errorf("invalid private key")
	}
	return secp.PrivKeyFromBytes(data).ToECDSA(), nil
}
func HexToECDSA(s string) (*ecdsa.PrivateKey, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil, err
	}
	return ToECDSA(b)
}

func Sign(hash []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash is required to be exactly 32 bytes")
	}
	priv := make([]byte, 32)
	key.D.FillBytes(priv)
	compact := secpecdsa.SignCompact(secp.PrivKeyFromBytes(priv), hash, false)
	// Decred compact: header || R || S. Ethereum: R || S || recovery id.
	out := make([]byte, 65)
	copy(out[:64], compact[1:])
	out[64] = (compact[0] - 27) & 3
	if out[64] > 1 {
		return nil, fmt.Errorf("unsupported recovery id %d", out[64])
	}
	return out, nil
}

func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	if len(hash) != 32 || len(sig) != 65 || sig[64] > 1 {
		return nil, fmt.Errorf("invalid signature")
	}
	compact := make([]byte, 65)
	compact[0] = 27 + sig[64]
	copy(compact[1:], sig[:64])
	pub, _, err := secpecdsa.RecoverCompact(compact, hash)
	if err != nil {
		return nil, err
	}
	return pub.ToECDSA(), nil
}

func PubkeyToAddress(pub ecdsa.PublicKey) common.Address {
	x := pub.X.FillBytes(make([]byte, 32))
	y := pub.Y.FillBytes(make([]byte, 32))
	return common.BytesToAddress(Keccak256(x, y)[12:])
}

func CreateAddress(addr common.Address, nonce uint64) common.Address {
	payload := append(rlpBytes(addr[:]), rlpUint(nonce)...)
	return common.BytesToAddress(Keccak256(rlpList(payload))[12:])
}
func CreateAddress2(addr common.Address, salt [32]byte, initHash []byte) common.Address {
	return common.BytesToAddress(Keccak256([]byte{0xff}, addr[:], salt[:], initHash)[12:])
}

func rlpUint(v uint64) []byte {
	if v == 0 {
		return []byte{0x80}
	}
	b := new(big.Int).SetUint64(v).Bytes()
	return rlpBytes(b)
}
func rlpBytes(b []byte) []byte {
	if len(b) == 1 && b[0] < 0x80 {
		return append([]byte(nil), b...)
	}
	if len(b) <= 55 {
		return append([]byte{byte(0x80 + len(b))}, b...)
	}
	l := new(big.Int).SetInt64(int64(len(b))).Bytes()
	out := append([]byte{byte(0xb7 + len(l))}, l...)
	return append(out, b...)
}
func rlpList(b []byte) []byte {
	if len(b) <= 55 {
		return append([]byte{byte(0xc0 + len(b))}, b...)
	}
	l := new(big.Int).SetInt64(int64(len(b))).Bytes()
	out := append([]byte{byte(0xf7 + len(l))}, l...)
	return append(out, b...)
}
