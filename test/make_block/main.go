package main

import (
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

type dummyHasher struct{}

func (h *dummyHasher) Reset()                   {}
func (h *dummyHasher) Update(k, v []byte) error { return nil }
func (h *dummyHasher) Hash() common.Hash        { return common.Hash{} }

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: make_block <output_file>")
		os.Exit(1)
	}
	outFile := os.Args[1]

	// 1. Setup keys
	key, _ := crypto.HexToECDSA("4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318")
	addr := crypto.PubkeyToAddress(key.PublicKey)
	fmt.Printf("Sender: %s\n", addr.Hex())

	// 2. Create Transaction
	// Transfer 100 wei to 0x00...01
	to := common.HexToAddress("0x0000000000000000000000000000000000000001")
	tx := types.NewTransaction(
		1,               // nonce
		to,              // to
		big.NewInt(100), // value
		21000,           // gas
		big.NewInt(1),   // gasPrice
		nil,             // data
	)

	// 3. Sign Transaction (ChainID 1337)
	signer := types.NewEIP155Signer(big.NewInt(1337))
	signedTx, err := types.SignTx(tx, signer, key)
	if err != nil {
		panic(err)
	}

	// 4. Create Block
	header := &types.Header{
		ParentHash: common.Hash{},
		Number:     big.NewInt(1),
		Time:       1000,
		GasLimit:   1000000,
		Coinbase:   common.HexToAddress("0x9999999999999999999999999999999999999999"),
	}

	body := &types.Body{Transactions: []*types.Transaction{signedTx}}
	block := types.NewBlock(header, body, nil, &dummyHasher{})

	// 5. Encode to RLP
	data, err := rlp.EncodeToBytes(block)
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile(outFile, data, 0644); err != nil {
		panic(err)
	}
	fmt.Printf("Block written to %s\n", outFile)
}
