//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/crypto"
	"github.com/smallyunet/echoevm/internal/eth/hexutil"
	"github.com/smallyunet/echoevm/internal/eth/types"
	"github.com/smallyunet/echoevm/internal/replay"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run generate_fixture.go <output.json>")
		os.Exit(2)
	}
	key, err := crypto.HexToECDSA("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		panic(err)
	}
	sender := crypto.PubkeyToAddress(key.PublicKey)
	recipient := common.HexToAddress("0x2000000000000000000000000000000000000002")
	tx, err := types.SignTx(types.NewTransaction(0, recipient, new(big.Int), 21_000, big.NewInt(1), nil), types.NewEIP155Signer(big.NewInt(1)), key)
	if err != nil {
		panic(err)
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		panic(err)
	}
	header := types.Header{Number: big.NewInt(19_500_000), Time: 1710338135, GasLimit: 30_000_000, Difficulty: new(big.Int)}
	witness := replay.Witness{
		Schema: replay.WitnessSchemaVersion, ChainID: 1, BlockHash: header.Hash(), Header: header,
		Transaction: raw, Source: "EchoEVM Chrome Wasm test fixture",
		Prestate: map[string]replay.WitnessAccount{
			sender.Hex():    {Balance: (*hexutil.Big)(big.NewInt(1_000_000)), Storage: map[common.Hash]common.Hash{}},
			recipient.Hex(): {Balance: (*hexutil.Big)(new(big.Int)), Storage: map[common.Hash]common.Hash{}},
		},
	}
	data, err := json.MarshalIndent(witness, "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(os.Args[1], data, 0o600); err != nil {
		panic(err)
	}
}
