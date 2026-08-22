package hexutil

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

type Bytes []byte
type Big big.Int
type Uint64 uint64

func Encode(b []byte) string       { return "0x" + hex.EncodeToString(b) }
func EncodeUint64(v uint64) string { return fmt.Sprintf("0x%x", v) }

func Decode(s string) ([]byte, error) {
	if !strings.HasPrefix(s, "0x") {
		return nil, fmt.Errorf("hex string without 0x prefix")
	}
	s = s[2:]
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("hex string has odd length")
	}
	return hex.DecodeString(s)
}

func (b Bytes) MarshalText() ([]byte, error) { return []byte(Encode(b)), nil }
func (b *Bytes) UnmarshalText(text []byte) error {
	raw, err := Decode(string(text))
	if err == nil {
		*b = raw
	}
	return err
}
func (b Bytes) MarshalJSON() ([]byte, error) { return json.Marshal(Encode(b)) }
func (b *Bytes) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return b.UnmarshalText([]byte(s))
}

func (u Uint64) MarshalText() ([]byte, error) { return []byte(EncodeUint64(uint64(u))), nil }
func (u *Uint64) UnmarshalText(text []byte) error {
	v, err := parseUint(string(text))
	if err == nil {
		*u = Uint64(v)
	}
	return err
}
func (u Uint64) MarshalJSON() ([]byte, error) { return json.Marshal(EncodeUint64(uint64(u))) }
func (u *Uint64) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return u.UnmarshalText([]byte(s))
}

func (b *Big) ToInt() *big.Int             { return (*big.Int)(b) }
func (b Big) MarshalText() ([]byte, error) { return []byte("0x" + (*big.Int)(&b).Text(16)), nil }
func (b *Big) UnmarshalText(text []byte) error {
	s := strings.TrimPrefix(string(text), "0x")
	if s == "" {
		s = "0"
	}
	v, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return fmt.Errorf("invalid hex integer")
	}
	*b = Big(*v)
	return nil
}
func (b Big) MarshalJSON() ([]byte, error) {
	text, _ := b.MarshalText()
	return json.Marshal(string(text))
}
func (b *Big) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return b.UnmarshalText([]byte(s))
}

func parseUint(s string) (uint64, error) {
	if !strings.HasPrefix(s, "0x") {
		return 0, fmt.Errorf("hex quantity without 0x prefix")
	}
	s = s[2:]
	if s == "" {
		return 0, fmt.Errorf("empty hex quantity")
	}
	return strconv.ParseUint(s, 16, 64)
}
