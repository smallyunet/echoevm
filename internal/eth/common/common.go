// Package common contains the small fixed-size Ethereum value types used by
// EchoEVM. It is intentionally owned by EchoEVM so the execution kernel does
// not inherit an execution-client dependency for basic protocol primitives.
package common

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
)

const HashLength = 32

var Big0 = new(big.Int)

type Hash [HashLength]byte
type Address [20]byte

func BytesToHash(b []byte) Hash {
	var h Hash
	copy(h[HashLength-min(len(b), HashLength):], b[max(0, len(b)-HashLength):])
	return h
}
func BigToHash(b *big.Int) Hash {
	if b == nil {
		return Hash{}
	}
	return BytesToHash(b.Bytes())
}
func HexToHash(s string) Hash { b, _ := ParseHexOrString(s); return BytesToHash(b) }
func BytesToAddress(b []byte) Address {
	var a Address
	copy(a[20-min(len(b), 20):], b[max(0, len(b)-20):])
	return a
}
func BigToAddress(b *big.Int) Address {
	if b == nil {
		return Address{}
	}
	return BytesToAddress(b.Bytes())
}
func HexToAddress(s string) Address { b, _ := ParseHexOrString(s); return BytesToAddress(b) }

func (h Hash) Bytes() []byte     { return append([]byte(nil), h[:]...) }
func (h Hash) Hex() string       { return "0x" + hex.EncodeToString(h[:]) }
func (h Hash) String() string    { return h.Hex() }
func (h Hash) Big() *big.Int     { return new(big.Int).SetBytes(h[:]) }
func (a Address) Bytes() []byte  { return append([]byte(nil), a[:]...) }
func (a Address) Hex() string    { return checksumAddress(a) }
func (a Address) String() string { return a.Hex() }
func (a Address) Big() *big.Int  { return new(big.Int).SetBytes(a[:]) }

func (h Hash) MarshalText() ([]byte, error)    { return []byte(h.Hex()), nil }
func (a Address) MarshalText() ([]byte, error) { return []byte(a.Hex()), nil }
func (h *Hash) UnmarshalText(text []byte) error {
	b, err := decodeFixed(text, HashLength)
	if err == nil {
		copy(h[:], b)
	}
	return err
}
func (a *Address) UnmarshalText(text []byte) error {
	b, err := decodeFixed(text, 20)
	if err == nil {
		copy(a[:], b)
	}
	return err
}
func (h Hash) MarshalJSON() ([]byte, error)    { return json.Marshal(h.Hex()) }
func (a Address) MarshalJSON() ([]byte, error) { return json.Marshal(a.Hex()) }
func (h *Hash) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return h.UnmarshalText([]byte(s))
}
func (a *Address) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return a.UnmarshalText([]byte(s))
}

func IsHexAddress(s string) bool { _, err := decodeFixed([]byte(s), 20); return err == nil }
func IsHexHash(s string) bool    { _, err := decodeFixed([]byte(s), HashLength); return err == nil }

func ParseHexOrString(s string) ([]byte, error) {
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return []byte(s), nil
	}
	s = s[2:]
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

func LeftPadBytes(b []byte, n int) []byte {
	if len(b) >= n {
		return b
	}
	out := make([]byte, n)
	copy(out[n-len(b):], b)
	return out
}
func RightPadBytes(b []byte, n int) []byte {
	if len(b) >= n {
		return b
	}
	out := make([]byte, n)
	copy(out, b)
	return out
}

func decodeFixed(text []byte, n int) ([]byte, error) {
	s := strings.TrimPrefix(strings.TrimPrefix(string(text), "0x"), "0X")
	if len(s) != n*2 {
		return nil, errors.New("invalid fixed-length hex value")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func checksumAddress(a Address) string {
	// Lower-case output is canonical for execution and JSON equality. Checksum
	// presentation belongs at UI boundaries and must not affect protocol values.
	return "0x" + hex.EncodeToString(a[:])
}
