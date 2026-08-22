package types

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/holiman/uint256"
	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/crypto"
	"github.com/smallyunet/echoevm/internal/eth/hexutil"
	"github.com/smallyunet/echoevm/internal/eth/rlp"
)

const (
	LegacyTxType uint8 = iota
	AccessListTxType
	DynamicFeeTxType
	BlobTxType
	SetCodeTxType
)

const blobGasPerBlob = 131072

type AccessTuple struct {
	Address     common.Address
	StorageKeys []common.Hash
}
type AccessList []AccessTuple

type LegacyTx struct {
	Nonce    uint64
	GasPrice *big.Int
	Gas      uint64
	To       *common.Address
	Value    *big.Int
	Data     []byte
	V, R, S  *big.Int
}
type DynamicFeeTx struct {
	ChainID              *big.Int
	Nonce                uint64
	GasTipCap, GasFeeCap *big.Int
	Gas                  uint64
	To                   *common.Address
	Value                *big.Int
	Data                 []byte
	AccessList           AccessList
	V, R, S              *big.Int
}
type BlobTx struct {
	ChainID, GasTipCap, GasFeeCap *uint256.Int
	Nonce                         uint64
	Gas                           uint64
	To                            common.Address
	Value                         *uint256.Int
	Data                          []byte
	AccessList                    AccessList
	BlobFeeCap                    *uint256.Int
	BlobHashes                    []common.Hash
	V, R, S                       *uint256.Int
}
type SetCodeTx struct {
	ChainID, GasTipCap, GasFeeCap *uint256.Int
	Nonce                         uint64
	Gas                           uint64
	To                            common.Address
	Value                         *uint256.Int
	Data                          []byte
	AccessList                    AccessList
	AuthList                      []SetCodeAuthorization
	V, R, S                       *uint256.Int
}

type Transaction struct {
	typeID                                                     uint8
	nonce                                                      uint64
	gas                                                        uint64
	to                                                         *common.Address
	value, gasPrice, gasTipCap, gasFeeCap, blobFeeCap, chainID *big.Int
	data                                                       []byte
	accessList                                                 AccessList
	blobHashes                                                 []common.Hash
	authList                                                   []SetCodeAuthorization
	v, r, s                                                    *big.Int
}

func NewTransaction(nonce uint64, to common.Address, value *big.Int, gas uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newLegacy(nonce, &to, value, gas, gasPrice, data)
}
func NewContractCreation(nonce uint64, value *big.Int, gas uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newLegacy(nonce, nil, value, gas, gasPrice, data)
}
func newLegacy(nonce uint64, to *common.Address, value *big.Int, gas uint64, gasPrice *big.Int, data []byte) *Transaction {
	return &Transaction{typeID: LegacyTxType, nonce: nonce, to: to, value: copyBig(value), gas: gas, gasPrice: copyBig(gasPrice), data: append([]byte(nil), data...), v: new(big.Int), r: new(big.Int), s: new(big.Int), chainID: new(big.Int)}
}

func NewTx(inner any) *Transaction {
	switch x := inner.(type) {
	case *LegacyTx:
		tx := newLegacy(x.Nonce, x.To, x.Value, x.Gas, x.GasPrice, x.Data)
		tx.v, tx.r, tx.s = copyBig(x.V), copyBig(x.R), copyBig(x.S)
		return tx
	case *DynamicFeeTx:
		return &Transaction{typeID: DynamicFeeTxType, chainID: copyBig(x.ChainID), nonce: x.Nonce, gas: x.Gas, to: x.To, value: copyBig(x.Value), gasTipCap: copyBig(x.GasTipCap), gasFeeCap: copyBig(x.GasFeeCap), data: append([]byte(nil), x.Data...), accessList: cloneAccessList(x.AccessList), v: copyBig(x.V), r: copyBig(x.R), s: copyBig(x.S)}
	case *BlobTx:
		to := x.To
		return &Transaction{typeID: BlobTxType, chainID: u256Big(x.ChainID), nonce: x.Nonce, gas: x.Gas, to: &to, value: u256Big(x.Value), gasTipCap: u256Big(x.GasTipCap), gasFeeCap: u256Big(x.GasFeeCap), data: append([]byte(nil), x.Data...), accessList: cloneAccessList(x.AccessList), blobFeeCap: u256Big(x.BlobFeeCap), blobHashes: append([]common.Hash(nil), x.BlobHashes...), v: u256Big(x.V), r: u256Big(x.R), s: u256Big(x.S)}
	case *SetCodeTx:
		to := x.To
		return &Transaction{typeID: SetCodeTxType, chainID: u256Big(x.ChainID), nonce: x.Nonce, gas: x.Gas, to: &to, value: u256Big(x.Value), gasTipCap: u256Big(x.GasTipCap), gasFeeCap: u256Big(x.GasFeeCap), data: append([]byte(nil), x.Data...), accessList: cloneAccessList(x.AccessList), authList: append([]SetCodeAuthorization(nil), x.AuthList...), v: u256Big(x.V), r: u256Big(x.R), s: u256Big(x.S)}
	default:
		panic(fmt.Sprintf("unsupported transaction data %T", inner))
	}
}

func (t *Transaction) Type() uint8   { return t.typeID }
func (t *Transaction) Nonce() uint64 { return t.nonce }
func (t *Transaction) Gas() uint64   { return t.gas }
func (t *Transaction) To() *common.Address {
	if t.to == nil {
		return nil
	}
	v := *t.to
	return &v
}
func (t *Transaction) Value() *big.Int { return copyBig(t.value) }
func (t *Transaction) GasPrice() *big.Int {
	if t.typeID >= DynamicFeeTxType {
		return copyBig(t.gasFeeCap)
	}
	return copyBig(t.gasPrice)
}
func (t *Transaction) GasTipCap() *big.Int { return copyBig(t.gasTipCap) }
func (t *Transaction) GasFeeCap() *big.Int {
	if t.typeID >= DynamicFeeTxType {
		return copyBig(t.gasFeeCap)
	}
	return copyBig(t.gasPrice)
}
func (t *Transaction) Data() []byte              { return append([]byte(nil), t.data...) }
func (t *Transaction) AccessList() AccessList    { return cloneAccessList(t.accessList) }
func (t *Transaction) BlobHashes() []common.Hash { return append([]common.Hash(nil), t.blobHashes...) }
func (t *Transaction) SetCodeAuthorizations() []SetCodeAuthorization {
	return append([]SetCodeAuthorization(nil), t.authList...)
}
func (t *Transaction) BlobGas() uint64         { return uint64(len(t.blobHashes)) * blobGasPerBlob }
func (t *Transaction) BlobGasFeeCap() *big.Int { return copyBig(t.blobFeeCap) }
func (t *Transaction) ChainId() *big.Int       { return copyBig(t.chainID) }
func (t *Transaction) Protected() bool {
	return t.typeID != LegacyTxType || t.v.Sign() != 0 && (t.v.Uint64() != 27 && t.v.Uint64() != 28)
}
func (t *Transaction) EffectiveGasTipValue(base *big.Int) *big.Int {
	tip := copyBig(t.gasTipCap)
	if tip == nil {
		tip = new(big.Int)
	}
	room := new(big.Int).Sub(copyBig(t.gasFeeCap), base)
	if room.Sign() < 0 {
		return new(big.Int)
	}
	if tip.Cmp(room) > 0 {
		return room
	}
	return tip
}
func (t *Transaction) Hash() common.Hash { b, _ := t.MarshalBinary(); return crypto.Keccak256Hash(b) }

func (t *Transaction) MarshalBinary() ([]byte, error) {
	fields := t.fields(true)
	enc := rlp.EncodeList(fields...)
	if t.typeID == LegacyTxType {
		return enc, nil
	}
	return append([]byte{t.typeID}, enc...), nil
}
func (t *Transaction) UnmarshalBinary(b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("empty transaction")
	}
	typ := LegacyTxType
	payload := b
	if b[0] < 0x80 {
		typ = b[0]
		payload = b[1:]
	}
	if typ > SetCodeTxType {
		return fmt.Errorf("unsupported transaction type %d", typ)
	}
	elems, err := rlp.SplitList(payload)
	if err != nil {
		return err
	}
	return t.decodeFields(typ, elems)
}

func (t *Transaction) fields(signature bool) [][]byte {
	u := func(n uint64) []byte { return rlp.EncodeBytes(uintBytes(n)) }
	bi := func(n *big.Int) []byte {
		if n == nil {
			return rlp.EncodeBytes(nil)
		}
		return rlp.EncodeBytes(n.Bytes())
	}
	addr := func(a *common.Address) []byte {
		if a == nil {
			return rlp.EncodeBytes(nil)
		}
		return rlp.EncodeBytes(a[:])
	}
	access := encodeAccessList(t.accessList)
	var f [][]byte
	switch t.typeID {
	case LegacyTxType:
		f = [][]byte{u(t.nonce), bi(t.gasPrice), u(t.gas), addr(t.to), bi(t.value), rlp.EncodeBytes(t.data)}
	case AccessListTxType:
		f = [][]byte{bi(t.chainID), u(t.nonce), bi(t.gasPrice), u(t.gas), addr(t.to), bi(t.value), rlp.EncodeBytes(t.data), access}
	case DynamicFeeTxType:
		f = [][]byte{bi(t.chainID), u(t.nonce), bi(t.gasTipCap), bi(t.gasFeeCap), u(t.gas), addr(t.to), bi(t.value), rlp.EncodeBytes(t.data), access}
	case BlobTxType:
		f = [][]byte{bi(t.chainID), u(t.nonce), bi(t.gasTipCap), bi(t.gasFeeCap), u(t.gas), addr(t.to), bi(t.value), rlp.EncodeBytes(t.data), access, bi(t.blobFeeCap), encodeHashes(t.blobHashes)}
	case SetCodeTxType:
		f = [][]byte{bi(t.chainID), u(t.nonce), bi(t.gasTipCap), bi(t.gasFeeCap), u(t.gas), addr(t.to), bi(t.value), rlp.EncodeBytes(t.data), access, encodeAuthorizations(t.authList)}
	}
	if signature {
		return append(f, bi(t.v), bi(t.r), bi(t.s))
	}
	return f
}

func (t *Transaction) decodeFields(typ uint8, e []rlp.RawValue) error {
	unsigned := map[uint8]int{LegacyTxType: 6, AccessListTxType: 8, DynamicFeeTxType: 9, BlobTxType: 11, SetCodeTxType: 10}[typ]
	if len(e) != unsigned+3 {
		return fmt.Errorf("transaction type %d has %d fields, want %d", typ, len(e), unsigned+3)
	}
	get := func(i int) ([]byte, error) { return rlp.Bytes(e[i]) }
	num := func(i int) (*big.Int, error) { b, err := get(i); return new(big.Int).SetBytes(b), err }
	u := func(i int) (uint64, error) {
		n, err := num(i)
		if err != nil {
			return 0, err
		}
		if !n.IsUint64() {
			return 0, fmt.Errorf("integer overflow")
		}
		return n.Uint64(), nil
	}
	to := func(i int) (*common.Address, error) {
		b, err := get(i)
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			return nil, nil
		}
		if len(b) != 20 {
			return nil, fmt.Errorf("invalid to length")
		}
		a := common.BytesToAddress(b)
		return &a, nil
	}
	*t = Transaction{typeID: typ, v: new(big.Int), r: new(big.Int), s: new(big.Int), chainID: new(big.Int), value: new(big.Int), gasPrice: new(big.Int), gasTipCap: new(big.Int), gasFeeCap: new(big.Int), blobFeeCap: new(big.Int)}
	var err error
	readBytes := func(index int) []byte {
		if err != nil {
			return nil
		}
		var value []byte
		value, err = get(index)
		return value
	}
	readNumber := func(index int) *big.Int {
		if err != nil {
			return new(big.Int)
		}
		var value *big.Int
		value, err = num(index)
		return value
	}
	readUint64 := func(index int) uint64 {
		if err != nil {
			return 0
		}
		var value uint64
		value, err = u(index)
		return value
	}
	readTo := func(index int) *common.Address {
		if err != nil {
			return nil
		}
		var value *common.Address
		value, err = to(index)
		return value
	}
	readAccessList := func(index int) AccessList {
		if err != nil {
			return nil
		}
		var value AccessList
		value, err = decodeAccessList(e[index])
		return value
	}
	switch typ {
	case LegacyTxType:
		t.nonce = readUint64(0)
		t.gasPrice = readNumber(1)
		t.gas = readUint64(2)
		t.to = readTo(3)
		t.value = readNumber(4)
		t.data = readBytes(5)
	case AccessListTxType:
		t.chainID = readNumber(0)
		t.nonce = readUint64(1)
		t.gasPrice = readNumber(2)
		t.gas = readUint64(3)
		t.to = readTo(4)
		t.value = readNumber(5)
		t.data = readBytes(6)
		t.accessList = readAccessList(7)
	case DynamicFeeTxType:
		t.chainID = readNumber(0)
		t.nonce = readUint64(1)
		t.gasTipCap = readNumber(2)
		t.gasFeeCap = readNumber(3)
		t.gas = readUint64(4)
		t.to = readTo(5)
		t.value = readNumber(6)
		t.data = readBytes(7)
		t.accessList = readAccessList(8)
	case BlobTxType:
		t.chainID = readNumber(0)
		t.nonce = readUint64(1)
		t.gasTipCap = readNumber(2)
		t.gasFeeCap = readNumber(3)
		t.gas = readUint64(4)
		t.to = readTo(5)
		t.value = readNumber(6)
		t.data = readBytes(7)
		t.accessList = readAccessList(8)
		t.blobFeeCap = readNumber(9)
		if err == nil {
			t.blobHashes, err = decodeHashes(e[10])
		}
	case SetCodeTxType:
		t.chainID = readNumber(0)
		t.nonce = readUint64(1)
		t.gasTipCap = readNumber(2)
		t.gasFeeCap = readNumber(3)
		t.gas = readUint64(4)
		t.to = readTo(5)
		t.value = readNumber(6)
		t.data = readBytes(7)
		t.accessList = readAccessList(8)
		if err == nil {
			t.authList, err = decodeAuthorizations(e[9])
		}
	}
	if err != nil {
		return err
	}
	t.v = readNumber(unsigned)
	t.r = readNumber(unsigned + 1)
	t.s = readNumber(unsigned + 2)
	if err != nil {
		return err
	}
	if typ == LegacyTxType && t.Protected() {
		t.chainID = deriveChainID(t.v)
	}
	return nil
}

type Signer struct {
	chainID   *big.Int
	homestead bool
}

func NewEIP155Signer(id *big.Int) Signer            { return Signer{chainID: copyBig(id)} }
func HomesteadSigner() Signer                       { return Signer{homestead: true} }
func LatestSignerForChainID(id *big.Int) Signer     { return NewEIP155Signer(id) }
func MakeSigner(_ any, _ *big.Int, _ uint64) Signer { return NewEIP155Signer(big.NewInt(1)) }
func SignTx(tx *Transaction, signer Signer, key *ecdsa.PrivateKey) (*Transaction, error) {
	cp := *tx
	cp.chainID = copyBig(signer.chainID)
	hash := signingHash(&cp, signer)
	sig, err := crypto.Sign(hash[:], key)
	if err != nil {
		return nil, err
	}
	cp.r = new(big.Int).SetBytes(sig[:32])
	cp.s = new(big.Int).SetBytes(sig[32:64])
	if cp.typeID == LegacyTxType {
		if signer.homestead || signer.chainID == nil {
			cp.v = new(big.Int).SetUint64(uint64(sig[64]) + 27)
		} else {
			cp.v = new(big.Int).Add(new(big.Int).Mul(signer.chainID, big.NewInt(2)), new(big.Int).SetUint64(uint64(sig[64])+35))
		}
	} else {
		cp.v = new(big.Int).SetUint64(uint64(sig[64]))
	}
	return &cp, nil
}
func Sender(signer Signer, tx *Transaction) (common.Address, error) {
	chain := tx.ChainId()
	if signer.chainID != nil && chain.Sign() != 0 && chain.Cmp(signer.chainID) != 0 {
		return common.Address{}, fmt.Errorf("invalid chain id")
	}
	hash := signingHash(tx, Signer{chainID: chain, homestead: !tx.Protected()})
	rec := byte(0)
	if tx.typeID == LegacyTxType {
		if tx.Protected() {
			rec = byte(new(big.Int).Sub(tx.v, new(big.Int).Add(new(big.Int).Mul(chain, big.NewInt(2)), big.NewInt(35))).Uint64())
		} else {
			rec = byte(tx.v.Uint64() - 27)
		}
	} else {
		rec = byte(tx.v.Uint64())
	}
	sig := make([]byte, 65)
	tx.r.FillBytes(sig[:32])
	tx.s.FillBytes(sig[32:64])
	sig[64] = rec
	pub, err := crypto.SigToPub(hash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pub), nil
}
func signingHash(t *Transaction, s Signer) common.Hash {
	f := t.fields(false)
	if t.typeID == LegacyTxType && s.chainID != nil && !s.homestead {
		f = append(f, rlp.EncodeBytes(s.chainID.Bytes()), rlp.EncodeBytes(nil), rlp.EncodeBytes(nil))
	}
	enc := rlp.EncodeList(f...)
	if t.typeID != LegacyTxType {
		enc = append([]byte{t.typeID}, enc...)
	}
	return crypto.Keccak256Hash(enc)
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	to := any(nil)
	if t.to != nil {
		to = t.to.Hex()
	}
	m := map[string]any{"type": hexutil.EncodeUint64(uint64(t.typeID)), "nonce": hexutil.EncodeUint64(t.nonce), "gas": hexutil.EncodeUint64(t.gas), "to": to, "value": "0x" + t.value.Text(16), "input": hexutil.Encode(t.data), "hash": t.Hash().Hex(), "v": "0x" + t.v.Text(16), "r": "0x" + t.r.Text(16), "s": "0x" + t.s.Text(16)}
	if t.typeID >= DynamicFeeTxType {
		m["maxPriorityFeePerGas"] = "0x" + t.gasTipCap.Text(16)
		m["maxFeePerGas"] = "0x" + t.gasFeeCap.Text(16)
	} else {
		m["gasPrice"] = "0x" + t.gasPrice.Text(16)
	}
	return json.Marshal(m)
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type, Nonce, Gas, GasPrice, MaxPriorityFeePerGas, MaxFeePerGas, Value, Input, ChainID, V, R, S string
		To                                                                                             *common.Address        `json:"to"`
		AccessList                                                                                     AccessList             `json:"accessList"`
		MaxFeePerBlobGas                                                                               string                 `json:"maxFeePerBlobGas"`
		BlobVersionedHashes                                                                            []common.Hash          `json:"blobVersionedHashes"`
		AuthorizationList                                                                              []SetCodeAuthorization `json:"authorizationList"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	q := func(s string) (*big.Int, error) {
		if s == "" {
			return new(big.Int), nil
		}
		n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
		if !ok {
			return nil, fmt.Errorf("invalid quantity %q", s)
		}
		return n, nil
	}
	typ, err := q(raw.Type)
	if err != nil {
		return err
	}
	nonce, err := q(raw.Nonce)
	if err != nil || !nonce.IsUint64() {
		return fmt.Errorf("invalid nonce")
	}
	gas, err := q(raw.Gas)
	if err != nil || !gas.IsUint64() {
		return fmt.Errorf("invalid gas")
	}
	value, err := q(raw.Value)
	if err != nil {
		return err
	}
	input, err := hexutil.Decode(raw.Input)
	if err != nil && raw.Input != "" {
		return err
	}
	*t = Transaction{typeID: uint8(typ.Uint64()), nonce: nonce.Uint64(), gas: gas.Uint64(), to: raw.To, value: value, data: input, accessList: cloneAccessList(raw.AccessList), blobHashes: append([]common.Hash(nil), raw.BlobVersionedHashes...), authList: append([]SetCodeAuthorization(nil), raw.AuthorizationList...)}
	t.gasPrice, _ = q(raw.GasPrice)
	t.gasTipCap, _ = q(raw.MaxPriorityFeePerGas)
	t.gasFeeCap, _ = q(raw.MaxFeePerGas)
	t.blobFeeCap, _ = q(raw.MaxFeePerBlobGas)
	t.chainID, _ = q(raw.ChainID)
	t.v, _ = q(raw.V)
	t.r, _ = q(raw.R)
	t.s, _ = q(raw.S)
	if t.typeID == LegacyTxType && t.chainID.Sign() == 0 && t.Protected() {
		t.chainID = deriveChainID(t.v)
	}
	return nil
}

func copyBig(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(v)
}
func u256Big(v *uint256.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}
	return v.ToBig()
}
func uintBytes(v uint64) []byte {
	if v == 0 {
		return nil
	}
	return new(big.Int).SetUint64(v).Bytes()
}
func cloneAccessList(v AccessList) AccessList {
	out := make(AccessList, len(v))
	for i, x := range v {
		out[i] = AccessTuple{Address: x.Address, StorageKeys: append([]common.Hash(nil), x.StorageKeys...)}
	}
	return out
}
func deriveChainID(v *big.Int) *big.Int {
	if v == nil || v.Cmp(big.NewInt(35)) < 0 {
		return new(big.Int)
	}
	return new(big.Int).Div(new(big.Int).Sub(v, big.NewInt(35)), big.NewInt(2))
}

func encodeAccessList(list AccessList) []byte {
	entries := make([][]byte, len(list))
	for i, e := range list {
		keys := make([][]byte, len(e.StorageKeys))
		for j, k := range e.StorageKeys {
			keys[j] = rlp.EncodeBytes(k[:])
		}
		entries[i] = rlp.EncodeList(rlp.EncodeBytes(e.Address[:]), rlp.EncodeList(keys...))
	}
	return rlp.EncodeList(entries...)
}
func decodeAccessList(raw rlp.RawValue) (AccessList, error) {
	entries, err := rlp.SplitList(raw)
	if err != nil {
		return nil, err
	}
	out := make(AccessList, len(entries))
	for i, e := range entries {
		pair, err := rlp.SplitList(e)
		if err != nil || len(pair) != 2 {
			return nil, fmt.Errorf("invalid access tuple")
		}
		a, _ := rlp.Bytes(pair[0])
		out[i].Address = common.BytesToAddress(a)
		keys, err := rlp.SplitList(pair[1])
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			b, _ := rlp.Bytes(k)
			out[i].StorageKeys = append(out[i].StorageKeys, common.BytesToHash(b))
		}
	}
	return out, nil
}
func encodeHashes(h []common.Hash) []byte {
	v := make([][]byte, len(h))
	for i, x := range h {
		v[i] = rlp.EncodeBytes(x[:])
	}
	return rlp.EncodeList(v...)
}
func decodeHashes(raw rlp.RawValue) ([]common.Hash, error) {
	v, err := rlp.SplitList(raw)
	if err != nil {
		return nil, err
	}
	out := make([]common.Hash, len(v))
	for i, x := range v {
		b, err := rlp.Bytes(x)
		if err != nil || len(b) != 32 {
			return nil, fmt.Errorf("invalid hash")
		}
		out[i] = common.BytesToHash(b)
	}
	return out, nil
}
