package core

import (
	"fmt"
	"math/big"
	"strings"
)

// Fork names
const (
	ForkFrontier       = "Frontier"
	ForkHomestead      = "Homestead"
	ForkTangerine      = "TangerineWhistle"
	ForkSpuriousDragon = "SpuriousDragon"
	ForkByzantium      = "Byzantium"
	ForkConstantinople = "Constantinople"
	ForkPetersburg     = "Petersburg"
	ForkIstanbul       = "Istanbul"
	ForkBerlin         = "Berlin"
	ForkLondon         = "London"
	ForkParis          = "Paris" // The Merge
	ForkShanghai       = "Shanghai"
	ForkCancun         = "Cancun"
	ForkPrague         = "Prague"
	ForkOsaka          = "Osaka"
)

var SupportedForks = []string{
	ForkFrontier, ForkHomestead, ForkTangerine, ForkSpuriousDragon,
	ForkByzantium, ForkConstantinople, ForkPetersburg, ForkIstanbul,
	ForkBerlin, ForkLondon, ForkParis, ForkShanghai, ForkCancun,
	ForkPrague, ForkOsaka,
}

func NormalizeFork(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	for _, fork := range SupportedForks {
		if strings.EqualFold(trimmed, fork) {
			return fork, nil
		}
	}
	return "", fmt.Errorf("unsupported fork %q; supported forks: %s", name, strings.Join(SupportedForks, ", "))
}

// ChainConfig holds chain configuration parameters and fork block numbers.
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"`

	HomesteadBlock      *big.Int `json:"homesteadBlock,omitempty"`
	EIP150Block         *big.Int `json:"eip150Block,omitempty"`
	EIP155Block         *big.Int `json:"eip155Block,omitempty"`
	EIP158Block         *big.Int `json:"eip158Block,omitempty"`
	ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"`
	PetersburgBlock     *big.Int `json:"petersburgBlock,omitempty"`
	IstanbulBlock       *big.Int `json:"istanbulBlock,omitempty"`
	BerlinBlock         *big.Int `json:"berlinBlock,omitempty"`
	LondonBlock         *big.Int `json:"londonBlock,omitempty"`
	ParisBlock          *big.Int `json:"parisBlock,omitempty"` // The Merge (PoS transition)
	ShanghaiBlock       *big.Int `json:"shanghaiBlock,omitempty"`
	CancunBlock         *big.Int `json:"cancunBlock,omitempty"`
	PragueBlock         *big.Int `json:"pragueBlock,omitempty"`
	OsakaBlock          *big.Int `json:"osakaBlock,omitempty"`
}

// Rules represents the active forks for a specific block number.
type Rules struct {
	ChainID          *big.Int
	IsHomestead      bool
	IsEIP150         bool
	IsEIP155         bool
	IsEIP158         bool
	IsByzantium      bool
	IsConstantinople bool
	IsPetersburg     bool
	IsIstanbul       bool
	IsBerlin         bool
	IsLondon         bool
	IsParis          bool
	IsShanghai       bool
	IsCancun         bool
	IsPrague         bool
	IsOsaka          bool
}

// Rules determines the active forks for the given block number.
func (c *ChainConfig) Rules(num *big.Int) Rules {
	chainID := c.ChainID
	if chainID == nil {
		chainID = new(big.Int)
	}

	return Rules{
		ChainID:          new(big.Int).Set(chainID),
		IsHomestead:      isForked(c.HomesteadBlock, num),
		IsEIP150:         isForked(c.EIP150Block, num),
		IsEIP155:         isForked(c.EIP155Block, num),
		IsEIP158:         isForked(c.EIP158Block, num),
		IsByzantium:      isForked(c.ByzantiumBlock, num),
		IsConstantinople: isForked(c.ConstantinopleBlock, num),
		IsPetersburg:     isForked(c.PetersburgBlock, num),
		IsIstanbul:       isForked(c.IstanbulBlock, num),
		IsBerlin:         isForked(c.BerlinBlock, num),
		IsLondon:         isForked(c.LondonBlock, num),
		IsParis:          isForked(c.ParisBlock, num),
		IsShanghai:       isForked(c.ShanghaiBlock, num),
		IsCancun:         isForked(c.CancunBlock, num),
		IsPrague:         isForked(c.PragueBlock, num),
		IsOsaka:          isForked(c.OsakaBlock, num),
	}
}

func isForked(forkBlock, currentBlock *big.Int) bool {
	if forkBlock == nil {
		return false
	}
	return currentBlock.Cmp(forkBlock) >= 0
}

// DefaultChainConfig returns a configuration with all forks active from block 0,
// suitable for testing behavior of the latest fork.
var DefaultChainConfig = &ChainConfig{
	ChainID:             big.NewInt(1),
	HomesteadBlock:      big.NewInt(0),
	EIP150Block:         big.NewInt(0),
	EIP155Block:         big.NewInt(0),
	EIP158Block:         big.NewInt(0),
	ByzantiumBlock:      big.NewInt(0),
	ConstantinopleBlock: big.NewInt(0),
	PetersburgBlock:     big.NewInt(0),
	IstanbulBlock:       big.NewInt(0),
	BerlinBlock:         big.NewInt(0),
	LondonBlock:         big.NewInt(0),
	ParisBlock:          big.NewInt(0),
	ShanghaiBlock:       big.NewInt(0),
	CancunBlock:         big.NewInt(0),
	PragueBlock:         big.NewInt(0),
	OsakaBlock:          big.NewInt(0),
}

func ChainConfigForFork(name string) (*ChainConfig, error) {
	fork, err := NormalizeFork(name)
	if err != nil {
		return nil, err
	}
	config := &ChainConfig{ChainID: big.NewInt(1)}
	// Activate a rule when the requested fork is at or after its introduction.
	forkIndex := 0
	for index, candidate := range SupportedForks {
		if candidate == fork {
			forkIndex = index
			break
		}
	}
	set := func(target string, field **big.Int) {
		for index, candidate := range SupportedForks {
			if candidate == target && forkIndex >= index {
				*field = big.NewInt(0)
			}
		}
	}
	set(ForkHomestead, &config.HomesteadBlock)
	set(ForkTangerine, &config.EIP150Block)
	set(ForkSpuriousDragon, &config.EIP155Block)
	set(ForkSpuriousDragon, &config.EIP158Block)
	set(ForkByzantium, &config.ByzantiumBlock)
	set(ForkConstantinople, &config.ConstantinopleBlock)
	set(ForkPetersburg, &config.PetersburgBlock)
	set(ForkIstanbul, &config.IstanbulBlock)
	set(ForkBerlin, &config.BerlinBlock)
	set(ForkLondon, &config.LondonBlock)
	set(ForkParis, &config.ParisBlock)
	set(ForkShanghai, &config.ShanghaiBlock)
	set(ForkCancun, &config.CancunBlock)
	set(ForkPrague, &config.PragueBlock)
	set(ForkOsaka, &config.OsakaBlock)
	return config, nil
}
