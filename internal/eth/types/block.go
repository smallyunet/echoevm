package types

import (
	"encoding/json"
	"math/big"

	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/crypto"
	"github.com/smallyunet/echoevm/internal/eth/hexutil"
	"github.com/smallyunet/echoevm/internal/eth/rlp"
)

var EmptyRootHash = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

type Header struct {
	ParentHash       common.Hash    `json:"parentHash"`
	UncleHash        common.Hash    `json:"sha3Uncles"`
	Coinbase         common.Address `json:"miner"`
	Root             common.Hash    `json:"stateRoot"`
	TxHash           common.Hash    `json:"transactionsRoot"`
	ReceiptHash      common.Hash    `json:"receiptsRoot"`
	Bloom            [256]byte      `json:"logsBloom"`
	Difficulty       *big.Int       `json:"difficulty"`
	Number           *big.Int       `json:"number"`
	GasLimit         uint64         `json:"gasLimit"`
	GasUsed          uint64         `json:"gasUsed"`
	Time             uint64         `json:"timestamp"`
	Extra            []byte         `json:"extraData"`
	MixDigest        common.Hash    `json:"mixHash"`
	Nonce            [8]byte        `json:"nonce"`
	BaseFee          *big.Int       `json:"baseFeePerGas,omitempty"`
	WithdrawalsHash  *common.Hash   `json:"withdrawalsRoot,omitempty"`
	BlobGasUsed      *uint64        `json:"blobGasUsed,omitempty"`
	ExcessBlobGas    *uint64        `json:"excessBlobGas,omitempty"`
	ParentBeaconRoot *common.Hash   `json:"parentBeaconBlockRoot,omitempty"`
	RequestsHash     *common.Hash   `json:"requestsHash,omitempty"`
}

func (h Header) MarshalJSON() ([]byte, error) {
	quantity := func(v *big.Int) *hexutil.Big {
		if v == nil {
			return nil
		}
		return (*hexutil.Big)(v)
	}
	u64 := func(v *uint64) *hexutil.Uint64 {
		if v == nil {
			return nil
		}
		q := hexutil.Uint64(*v)
		return &q
	}
	return json.Marshal(struct {
		ParentHash       common.Hash     `json:"parentHash"`
		UncleHash        common.Hash     `json:"sha3Uncles"`
		Coinbase         common.Address  `json:"miner"`
		Root             common.Hash     `json:"stateRoot"`
		TxHash           common.Hash     `json:"transactionsRoot"`
		ReceiptHash      common.Hash     `json:"receiptsRoot"`
		Bloom            hexutil.Bytes   `json:"logsBloom"`
		Difficulty       *hexutil.Big    `json:"difficulty"`
		Number           *hexutil.Big    `json:"number"`
		GasLimit         hexutil.Uint64  `json:"gasLimit"`
		GasUsed          hexutil.Uint64  `json:"gasUsed"`
		Time             hexutil.Uint64  `json:"timestamp"`
		Extra            hexutil.Bytes   `json:"extraData"`
		MixDigest        common.Hash     `json:"mixHash"`
		Nonce            hexutil.Bytes   `json:"nonce"`
		BaseFee          *hexutil.Big    `json:"baseFeePerGas,omitempty"`
		WithdrawalsHash  *common.Hash    `json:"withdrawalsRoot,omitempty"`
		BlobGasUsed      *hexutil.Uint64 `json:"blobGasUsed,omitempty"`
		ExcessBlobGas    *hexutil.Uint64 `json:"excessBlobGas,omitempty"`
		ParentBeaconRoot *common.Hash    `json:"parentBeaconBlockRoot,omitempty"`
		RequestsHash     *common.Hash    `json:"requestsHash,omitempty"`
	}{h.ParentHash, h.UncleHash, h.Coinbase, h.Root, h.TxHash, h.ReceiptHash, hexutil.Bytes(h.Bloom[:]), quantity(h.Difficulty), quantity(h.Number), hexutil.Uint64(h.GasLimit), hexutil.Uint64(h.GasUsed), hexutil.Uint64(h.Time), hexutil.Bytes(h.Extra), h.MixDigest, hexutil.Bytes(h.Nonce[:]), quantity(h.BaseFee), h.WithdrawalsHash, u64(h.BlobGasUsed), u64(h.ExcessBlobGas), h.ParentBeaconRoot, h.RequestsHash})
}

func (h Header) Hash() common.Hash {
	fields := [][]byte{rlp.EncodeBytes(h.ParentHash[:]), rlp.EncodeBytes(h.UncleHash[:]), rlp.EncodeBytes(h.Coinbase[:]), rlp.EncodeBytes(h.Root[:]), rlp.EncodeBytes(h.TxHash[:]), rlp.EncodeBytes(h.ReceiptHash[:]), rlp.EncodeBytes(h.Bloom[:]), encBig(h.Difficulty), encBig(h.Number), encUint(h.GasLimit), encUint(h.GasUsed), encUint(h.Time), rlp.EncodeBytes(h.Extra), rlp.EncodeBytes(h.MixDigest[:]), rlp.EncodeBytes(h.Nonce[:])}
	if h.BaseFee != nil {
		fields = append(fields, encBig(h.BaseFee))
	}
	if h.WithdrawalsHash != nil {
		fields = append(fields, rlp.EncodeBytes(h.WithdrawalsHash[:]))
	}
	if h.BlobGasUsed != nil {
		fields = append(fields, encUint(*h.BlobGasUsed))
	}
	if h.ExcessBlobGas != nil {
		fields = append(fields, encUint(*h.ExcessBlobGas))
	}
	if h.ParentBeaconRoot != nil {
		fields = append(fields, rlp.EncodeBytes(h.ParentBeaconRoot[:]))
	}
	if h.RequestsHash != nil {
		fields = append(fields, rlp.EncodeBytes(h.RequestsHash[:]))
	}
	return crypto.Keccak256Hash(rlp.EncodeList(fields...))
}

func (h *Header) UnmarshalJSON(data []byte) error {
	type raw struct {
		ParentHash       common.Hash     `json:"parentHash"`
		UncleHash        common.Hash     `json:"sha3Uncles"`
		Coinbase         common.Address  `json:"miner"`
		Root             common.Hash     `json:"stateRoot"`
		TxHash           common.Hash     `json:"transactionsRoot"`
		ReceiptHash      common.Hash     `json:"receiptsRoot"`
		Bloom            hexutil.Bytes   `json:"logsBloom"`
		Difficulty       *hexutil.Big    `json:"difficulty"`
		Number           *hexutil.Big    `json:"number"`
		GasLimit         hexutil.Uint64  `json:"gasLimit"`
		GasUsed          hexutil.Uint64  `json:"gasUsed"`
		Time             hexutil.Uint64  `json:"timestamp"`
		Extra            hexutil.Bytes   `json:"extraData"`
		MixDigest        common.Hash     `json:"mixHash"`
		Nonce            hexutil.Bytes   `json:"nonce"`
		BaseFee          *hexutil.Big    `json:"baseFeePerGas"`
		WithdrawalsHash  *common.Hash    `json:"withdrawalsRoot"`
		BlobGasUsed      *hexutil.Uint64 `json:"blobGasUsed"`
		ExcessBlobGas    *hexutil.Uint64 `json:"excessBlobGas"`
		ParentBeaconRoot *common.Hash    `json:"parentBeaconBlockRoot"`
		RequestsHash     *common.Hash    `json:"requestsHash"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	h.ParentHash = r.ParentHash
	h.UncleHash = r.UncleHash
	h.Coinbase = r.Coinbase
	h.Root = r.Root
	h.TxHash = r.TxHash
	h.ReceiptHash = r.ReceiptHash
	copy(h.Bloom[:], r.Bloom)
	if r.Difficulty != nil {
		h.Difficulty = (*big.Int)(r.Difficulty)
	}
	if r.Number != nil {
		h.Number = (*big.Int)(r.Number)
	}
	h.GasLimit = uint64(r.GasLimit)
	h.GasUsed = uint64(r.GasUsed)
	h.Time = uint64(r.Time)
	h.Extra = r.Extra
	h.MixDigest = r.MixDigest
	copy(h.Nonce[:], r.Nonce)
	if r.BaseFee != nil {
		h.BaseFee = (*big.Int)(r.BaseFee)
	}
	h.WithdrawalsHash = r.WithdrawalsHash
	if r.BlobGasUsed != nil {
		v := uint64(*r.BlobGasUsed)
		h.BlobGasUsed = &v
	}
	if r.ExcessBlobGas != nil {
		v := uint64(*r.ExcessBlobGas)
		h.ExcessBlobGas = &v
	}
	h.ParentBeaconRoot = r.ParentBeaconRoot
	h.RequestsHash = r.RequestsHash
	return nil
}

type Receipt struct {
	TxHash  common.Hash `json:"transactionHash"`
	Status  uint64      `json:"status"`
	GasUsed uint64      `json:"gasUsed"`
}

const (
	ReceiptStatusFailed     uint64 = 0
	ReceiptStatusSuccessful uint64 = 1
)

func (r *Receipt) UnmarshalJSON(data []byte) error {
	var x struct {
		TxHash  common.Hash    `json:"transactionHash"`
		Status  hexutil.Uint64 `json:"status"`
		GasUsed hexutil.Uint64 `json:"gasUsed"`
	}
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	r.TxHash = x.TxHash
	r.Status = uint64(x.Status)
	r.GasUsed = uint64(x.GasUsed)
	return nil
}

type Account struct {
	Code    []byte                      `json:"code"`
	Storage map[common.Hash]common.Hash `json:"storage"`
	Balance *big.Int                    `json:"balance"`
	Nonce   uint64                      `json:"nonce"`
}
type GenesisAlloc map[common.Address]Account

func (a *Account) UnmarshalJSON(data []byte) error {
	var encoded struct {
		Code    hexutil.Bytes     `json:"code"`
		Storage map[string]string `json:"storage"`
		Balance *hexutil.Big      `json:"balance"`
		Nonce   hexutil.Uint64    `json:"nonce"`
	}
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}
	a.Code = append([]byte(nil), encoded.Code...)
	a.Storage = make(map[common.Hash]common.Hash, len(encoded.Storage))
	for key, value := range encoded.Storage {
		a.Storage[common.HexToHash(key)] = common.HexToHash(value)
	}
	if encoded.Balance != nil {
		a.Balance = new(big.Int).Set((*big.Int)(encoded.Balance))
	}
	a.Nonce = uint64(encoded.Nonce)
	return nil
}

func encBig(v *big.Int) []byte {
	if v == nil {
		return rlp.EncodeBytes(nil)
	}
	return rlp.EncodeBytes(v.Bytes())
}
func encUint(v uint64) []byte { return rlp.EncodeBytes(uintBytes(v)) }
