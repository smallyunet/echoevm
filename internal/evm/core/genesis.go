package core

import (
	"encoding/json"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// GenesisAccount is the JSON representation of an account in the genesis file.
type GenesisAccount struct {
	Code       hexutil.Bytes               `json:"code,omitempty"`
	Storage    map[common.Hash]common.Hash `json:"storage,omitempty"`
	Balance    *hexutil.Big                `json:"balance,omitempty"`
	Nonce      uint64                      `json:"nonce,omitempty"`
	PrivateKey hexutil.Bytes               `json:"secretKey,omitempty"` // for tests
}

// Genesis represents the genesis definition.
type Genesis struct {
	Config     *ChainConfig                      `json:"config"`
	Nonce      uint64                            `json:"nonce"`
	Timestamp  uint64                            `json:"timestamp"`
	ExtraData  hexutil.Bytes                     `json:"extraData"`
	GasLimit   uint64                            `json:"gasLimit"`
	Difficulty *hexutil.Big                      `json:"difficulty"`
	Mixhash    common.Hash                       `json:"mixHash"`
	Coinbase   common.Address                    `json:"coinbase"`
	Alloc      map[common.Address]GenesisAccount `json:"alloc"`
	Number     uint64                            `json:"number"`
	GasUsed    uint64                            `json:"gasUsed"`
	ParentHash common.Hash                       `json:"parentHash"`
}

// ChainConfig is the core config which determines the blockchain settings.
// Simplified for now.
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"`
}

// LoadGenesis loads a genesis JSON file.
func LoadGenesis(path string) (*Genesis, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var genesis Genesis
	if err := json.NewDecoder(file).Decode(&genesis); err != nil {
		return nil, err
	}
	return &genesis, nil
}

// ToStateDB populates a StateDB with the genesis accounts.
func (g *Genesis) ToStateDB(db StateDB) error {
	for addr, account := range g.Alloc {
		db.CreateAccount(addr)
		if account.Balance != nil {
			db.AddBalance(addr, account.Balance.ToInt())
		}
		if account.Code != nil {
			db.SetCode(addr, account.Code)
		}
		if account.Nonce != 0 {
			db.SetNonce(addr, account.Nonce)
		}
		for key, value := range account.Storage {
			db.SetState(addr, key, value)
		}
	}
	return nil
}

// DefaultGenesis returns a basic genesis for testing.
func DefaultGenesis() *Genesis {
	return &Genesis{
		Config: &ChainConfig{ChainID: big.NewInt(1)},
		Alloc:  make(map[common.Address]GenesisAccount),
	}
}
