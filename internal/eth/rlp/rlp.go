// Package rlp implements Ethereum Recursive Length Prefix encoding without an
// execution-client dependency. The reflection surface is deliberately small
// and covers EchoEVM protocol structures and trie nodes.
package rlp

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"reflect"
)

type Kind byte

const (
	String Kind = iota
	List
)

type RawValue []byte
type Encoder interface{ EncodeRLP(io.Writer) error }

func Encode(w io.Writer, v any) error {
	b, err := EncodeToBytes(v)
	if err == nil {
		_, err = w.Write(b)
	}
	return err
}

func EncodeBytes(b []byte) []byte { var w bytes.Buffer; _ = writeBytes(&w, b); return w.Bytes() }
func EncodeList(values ...[]byte) []byte {
	var p bytes.Buffer
	for _, value := range values {
		p.Write(value)
	}
	var w bytes.Buffer
	_ = writeListPayload(&w, p.Bytes())
	return w.Bytes()
}
func SplitList(b []byte) ([]RawValue, error) {
	k, payload, rest, err := Split(b)
	if err != nil {
		return nil, err
	}
	if k != List || len(rest) != 0 {
		return nil, fmt.Errorf("rlp: expected one list")
	}
	return rawElements(payload)
}
func Bytes(raw RawValue) ([]byte, error) {
	k, payload, rest, err := Split(raw)
	if err != nil {
		return nil, err
	}
	if k != String || len(rest) != 0 {
		return nil, fmt.Errorf("rlp: expected string")
	}
	return append([]byte(nil), payload...), nil
}
func EncodeToBytes(v any) ([]byte, error) {
	var b bytes.Buffer
	if err := encode(&b, reflect.ValueOf(v)); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func encode(w *bytes.Buffer, v reflect.Value) error {
	if !v.IsValid() {
		w.WriteByte(0x80)
		return nil
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			w.WriteByte(0x80)
			return nil
		}
		return encode(w, v.Elem())
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			w.WriteByte(0x80)
			return nil
		}
		if enc, ok := v.Interface().(Encoder); ok {
			return enc.EncodeRLP(w)
		}
		return encode(w, v.Elem())
	}
	if v.CanInterface() {
		if enc, ok := v.Interface().(Encoder); ok {
			return enc.EncodeRLP(w)
		}
	}
	if v.Type() == reflect.TypeOf(big.Int{}) {
		n := v.Interface().(big.Int)
		return writeBytes(w, n.Bytes())
	}
	switch v.Kind() {
	case reflect.String:
		return writeBytes(w, []byte(v.String()))
	case reflect.Bool:
		if v.Bool() {
			return writeBytes(w, []byte{1})
		}
		return writeBytes(w, nil)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n := v.Uint()
		if n == 0 {
			return writeBytes(w, nil)
		}
		return writeBytes(w, new(big.Int).SetUint64(n).Bytes())
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return writeBytes(w, v.Bytes())
		}
		return writeList(w, v)
	case reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			b := make([]byte, v.Len())
			reflect.Copy(reflect.ValueOf(b), v)
			return writeBytes(w, b)
		}
		return writeList(w, v)
	case reflect.Struct:
		var payload bytes.Buffer
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).PkgPath != "" {
				continue
			}
			if err := encode(&payload, v.Field(i)); err != nil {
				return err
			}
		}
		return writeListPayload(w, payload.Bytes())
	default:
		return fmt.Errorf("rlp: unsupported type %s", v.Type())
	}
}

func writeList(w *bytes.Buffer, v reflect.Value) error {
	var p bytes.Buffer
	for i := 0; i < v.Len(); i++ {
		if err := encode(&p, v.Index(i)); err != nil {
			return err
		}
	}
	return writeListPayload(w, p.Bytes())
}
func writeBytes(w *bytes.Buffer, b []byte) error {
	if len(b) == 1 && b[0] < 0x80 {
		w.Write(b)
		return nil
	}
	writePrefix(w, 0x80, 0xb7, len(b))
	w.Write(b)
	return nil
}
func writeListPayload(w *bytes.Buffer, b []byte) error {
	writePrefix(w, 0xc0, 0xf7, len(b))
	w.Write(b)
	return nil
}
func writePrefix(w *bytes.Buffer, short, long byte, n int) {
	if n <= 55 {
		w.WriteByte(short + byte(n))
		return
	}
	l := new(big.Int).SetInt64(int64(n)).Bytes()
	w.WriteByte(long + byte(len(l)))
	w.Write(l)
}

func Split(b []byte) (Kind, []byte, []byte, error) {
	if len(b) == 0 {
		return String, nil, nil, io.ErrUnexpectedEOF
	}
	p := b[0]
	switch {
	case p < 0x80:
		return String, b[:1], b[1:], nil
	case p <= 0xb7:
		n := int(p - 0x80)
		return splitN(String, b, 1, n)
	case p <= 0xbf:
		ll := int(p - 0xb7)
		n, err := readLen(b, ll)
		if err != nil {
			return 0, nil, nil, err
		}
		return splitN(String, b, 1+ll, n)
	case p <= 0xf7:
		n := int(p - 0xc0)
		return splitN(List, b, 1, n)
	default:
		ll := int(p - 0xf7)
		n, err := readLen(b, ll)
		if err != nil {
			return 0, nil, nil, err
		}
		return splitN(List, b, 1+ll, n)
	}
}
func splitN(k Kind, b []byte, off, n int) (Kind, []byte, []byte, error) {
	if n < 0 || off+n > len(b) {
		return 0, nil, nil, io.ErrUnexpectedEOF
	}
	return k, b[off : off+n], b[off+n:], nil
}
func readLen(b []byte, ll int) (int, error) {
	if ll == 0 || 1+ll > len(b) {
		return 0, io.ErrUnexpectedEOF
	}
	n := 0
	for _, x := range b[1 : 1+ll] {
		n = n<<8 | int(x)
	}
	return n, nil
}

func DecodeBytes(b []byte, out any) error {
	k, p, rest, err := Split(b)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return fmt.Errorf("rlp: trailing data")
	}
	return decode(reflect.ValueOf(out), k, p, b)
}
func decode(dst reflect.Value, k Kind, payload, raw []byte) error {
	if dst.Kind() != reflect.Pointer || dst.IsNil() {
		return fmt.Errorf("rlp: non-pointer output")
	}
	dst = dst.Elem()
	if dst.Type() == reflect.TypeOf(RawValue{}) {
		dst.SetBytes(raw)
		return nil
	}
	if dst.Kind() == reflect.Pointer {
		if dst.Type().Elem() == reflect.TypeOf(big.Int{}) {
			n := new(big.Int).SetBytes(payload)
			dst.Set(reflect.ValueOf(n))
			return nil
		}
		dst.Set(reflect.New(dst.Type().Elem()))
		return decode(dst, k, payload, raw)
	}
	if dst.Type() == reflect.TypeOf(big.Int{}) {
		dst.Set(reflect.ValueOf(*new(big.Int).SetBytes(payload)))
		return nil
	}
	switch dst.Kind() {
	case reflect.Slice:
		if dst.Type().Elem().Kind() == reflect.Uint8 {
			if k != String {
				return fmt.Errorf("rlp: expected string")
			}
			dst.SetBytes(append([]byte(nil), payload...))
			return nil
		}
		if k != List {
			return fmt.Errorf("rlp: expected list")
		}
		elems, err := rawElements(payload)
		if err != nil {
			return err
		}
		s := reflect.MakeSlice(dst.Type(), len(elems), len(elems))
		for i, e := range elems {
			ek, ep, _, _ := Split(e)
			if err := decode(s.Index(i).Addr(), ek, ep, e); err != nil {
				return err
			}
		}
		dst.Set(s)
		return nil
	case reflect.Array:
		if dst.Type().Elem().Kind() != reflect.Uint8 || k != String || len(payload) != dst.Len() {
			return fmt.Errorf("rlp: invalid byte array")
		}
		reflect.Copy(dst, reflect.ValueOf(payload))
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if len(payload) > 8 {
			return fmt.Errorf("rlp: integer overflow")
		}
		dst.SetUint(new(big.Int).SetBytes(payload).Uint64())
		return nil
	case reflect.Struct:
		if k != List {
			return fmt.Errorf("rlp: expected struct list")
		}
		elems, err := rawElements(payload)
		if err != nil {
			return err
		}
		j := 0
		for i := 0; i < dst.NumField(); i++ {
			if dst.Type().Field(i).PkgPath != "" {
				continue
			}
			if j >= len(elems) {
				return io.ErrUnexpectedEOF
			}
			ek, ep, _, _ := Split(elems[j])
			if err := decode(dst.Field(i).Addr(), ek, ep, elems[j]); err != nil {
				return err
			}
			j++
		}
		return nil
	default:
		return fmt.Errorf("rlp: unsupported decode type %s", dst.Type())
	}
}
func rawElements(payload []byte) ([]RawValue, error) {
	var out []RawValue
	for len(payload) > 0 {
		_, _, rest, err := Split(payload)
		if err != nil {
			return nil, err
		}
		n := len(payload) - len(rest)
		out = append(out, append([]byte(nil), payload[:n]...))
		payload = rest
	}
	return out, nil
}
