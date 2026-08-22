package replay

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/crypto"
	"github.com/smallyunet/echoevm/internal/eth/hexutil"
	"github.com/smallyunet/echoevm/internal/eth/types"
	explaintrace "github.com/smallyunet/echoevm/internal/trace"
)

func TestReplayWitnessExecutesWithoutRPCComparison(t *testing.T) {
	witness, tx := signedTransferWitness(t)
	result, err := ReplayWitness(context.Background(), ReplayRequest{
		Witness: witness, Profile: explaintrace.ProfileAuto, Limit: DefaultEvidenceLimit,
		MaxMemoryBytes: DefaultEvidenceMemoryBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Transaction.Hash != tx.Hash().Hex() || result.Transaction.Status != "success" {
		t.Fatalf("transaction = %+v", result.Transaction)
	}
	if result.Execution.Engine != "EchoEVM" || result.Execution.GasUsed != 21_000 {
		t.Fatalf("execution = %+v", result.Execution)
	}
	if result.Evidence == nil || result.Evidence.Schema != explaintrace.EvidenceSchemaVersion {
		t.Fatalf("evidence = %+v", result.Evidence)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{`"geth"`, `"comparison"`, `"match"`, `"firstDivergence"`} {
		if bytes.Contains(encoded, []byte(forbidden)) {
			t.Fatalf("standalone replay leaked verification field %s: %s", forbidden, encoded)
		}
	}
	if result.Witness.Schema != WitnessSchemaVersion || len(result.Witness.SHA256) != 64 {
		t.Fatalf("witness provenance = %+v", result.Witness)
	}
}

func TestReplayWitnessRequiresSenderPrestate(t *testing.T) {
	witness, _ := signedTransferWitness(t)
	for address, account := range witness.Prestate {
		if account.Balance != nil && (*big.Int)(account.Balance).Sign() > 0 {
			delete(witness.Prestate, address)
		}
	}
	_, err := ReplayWitness(context.Background(), ReplayRequest{Witness: witness})
	if err == nil || !strings.Contains(err.Error(), "missing sender account") {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeWitnessRejectsUnknownFields(t *testing.T) {
	witness, _ := signedTransferWitness(t)
	data, err := json.Marshal(witness)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte(`"schema":`), []byte(`"unexpected":true,"schema":`), 1)
	_, err = DecodeWitness(bytes.NewReader(data))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error = %v", err)
	}
}

func signedTransferWitness(t *testing.T) (Witness, *types.Transaction) {
	t.Helper()
	key, err := crypto.HexToECDSA("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	sender := crypto.PubkeyToAddress(key.PublicKey)
	recipient := common.HexToAddress("0x2000000000000000000000000000000000000002")
	tx, err := types.SignTx(types.NewTransaction(0, recipient, new(big.Int), 21_000, big.NewInt(1), nil), types.NewEIP155Signer(big.NewInt(1)), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	header := types.Header{Number: big.NewInt(19_500_000), Time: 1710338135, GasLimit: 30_000_000, Difficulty: new(big.Int)}
	return Witness{
		Schema: WitnessSchemaVersion, ChainID: 1, BlockHash: header.Hash(), Header: header,
		Transaction: raw,
		Prestate: map[string]WitnessAccount{
			sender.Hex():    {Balance: (*hexutil.Big)(big.NewInt(1_000_000)), Storage: map[common.Hash]common.Hash{}},
			recipient.Hex(): {Balance: (*hexutil.Big)(new(big.Int)), Storage: map[common.Hash]common.Hash{}},
		},
	}, tx
}
