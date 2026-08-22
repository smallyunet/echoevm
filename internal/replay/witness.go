package replay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/hexutil"
	"github.com/smallyunet/echoevm/internal/eth/types"
)

const WitnessSchemaVersion = "echoevm.replay-witness.v1"
const maxWitnessBytes = 64 << 20

// Witness is the complete, versioned input required to replay one transaction
// without consulting another execution engine. Prestate must describe every
// account and storage slot touched by the transaction.
type Witness struct {
	Schema           string                    `json:"schema"`
	ChainID          uint64                    `json:"chainId"`
	BlockHash        common.Hash               `json:"blockHash"`
	TransactionIndex uint64                    `json:"transactionIndex"`
	Header           types.Header              `json:"header"`
	Transaction      hexutil.Bytes             `json:"transaction"`
	Prestate         map[string]WitnessAccount `json:"prestate"`
	BlockHashes      map[string]common.Hash    `json:"blockHashes,omitempty"`
	Source           string                    `json:"source,omitempty"`
}

type WitnessAccount struct {
	Balance *hexutil.Big                `json:"balance,omitempty"`
	Nonce   flexibleUint64              `json:"nonce"`
	Code    hexutil.Bytes               `json:"code,omitempty"`
	Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
}

type WitnessProvenance struct {
	Schema string `json:"schema"`
	SHA256 string `json:"sha256"`
	Source string `json:"source,omitempty"`
}

func LoadWitness(path string) (Witness, error) {
	file, err := os.Open(path)
	if err != nil {
		return Witness{}, fmt.Errorf("open replay witness: %w", err)
	}
	witness, decodeErr := DecodeWitness(file)
	closeErr := file.Close()
	if decodeErr != nil {
		return Witness{}, decodeErr
	}
	if closeErr != nil {
		return Witness{}, fmt.Errorf("close replay witness: %w", closeErr)
	}
	return witness, nil
}

func DecodeWitness(reader io.Reader) (Witness, error) {
	limited := io.LimitReader(reader, maxWitnessBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return Witness{}, fmt.Errorf("read replay witness: %w", err)
	}
	if len(data) > maxWitnessBytes {
		return Witness{}, fmt.Errorf("replay witness exceeds %d bytes", maxWitnessBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var witness Witness
	if err := decoder.Decode(&witness); err != nil {
		return Witness{}, fmt.Errorf("decode replay witness: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return Witness{}, err
	}
	if err := witness.Validate(); err != nil {
		return Witness{}, err
	}
	return witness, nil
}

func (w Witness) Validate() error {
	if w.Schema != WitnessSchemaVersion {
		return fmt.Errorf("unsupported replay witness schema %q: want %q", w.Schema, WitnessSchemaVersion)
	}
	if w.ChainID == 0 {
		return errors.New("replay witness chainId must be non-zero")
	}
	if w.Header.Number == nil {
		return errors.New("replay witness header is missing a block number")
	}
	if w.BlockHash == (common.Hash{}) || w.Header.Hash() != w.BlockHash {
		return errors.New("replay witness blockHash does not match its header")
	}
	if len(w.Transaction) == 0 {
		return errors.New("replay witness transaction is empty")
	}
	if len(w.Prestate) == 0 {
		return errors.New("replay witness prestate is empty")
	}
	for address := range w.Prestate {
		if !common.IsHexAddress(address) {
			return fmt.Errorf("replay witness contains invalid address %q", address)
		}
	}
	var tx types.Transaction
	if err := tx.UnmarshalBinary(w.Transaction); err != nil {
		return fmt.Errorf("decode replay witness transaction: %w", err)
	}
	if tx.Protected() && tx.ChainId().Cmp(new(big.Int).SetUint64(w.ChainID)) != 0 {
		return fmt.Errorf("replay witness transaction chainId %s does not match witness chainId %d", tx.ChainId(), w.ChainID)
	}
	return nil
}

func (w Witness) Provenance() (WitnessProvenance, error) {
	data, err := json.Marshal(w)
	if err != nil {
		return WitnessProvenance{}, fmt.Errorf("encode replay witness provenance: %w", err)
	}
	digest := sha256.Sum256(data)
	return WitnessProvenance{Schema: w.Schema, SHA256: hex.EncodeToString(digest[:]), Source: w.Source}, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("replay witness must contain exactly one JSON document")
		}
		return fmt.Errorf("decode replay witness: %w", err)
	}
	return nil
}
