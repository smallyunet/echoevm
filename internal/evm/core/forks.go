package core

import (
	"math/big"
)

// Fork names
const (
	ForkHomestead      = "Homestead"
	ForkByzantium      = "Byzantium"
	ForkConstantinople = "Constantinople"
	ForkIstanbul       = "Istanbul"
	ForkLondon         = "London"
	ForkParis          = "Paris" // The Merge
	ForkShanghai       = "Shanghai"
	ForkCancun         = "Cancun"
)

// ChainConfig holds chain configuration parameters and fork block numbers.
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"`

	HomesteadBlock      *big.Int `json:"homesteadBlock,omitempty"`
	ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"`
	IstanbulBlock       *big.Int `json:"istanbulBlock,omitempty"`
	LondonBlock         *big.Int `json:"londonBlock,omitempty"`
	ParisBlock          *big.Int `json:"parisBlock,omitempty"` // The Merge (PoS transition)
	ShanghaiBlock       *big.Int `json:"shanghaiBlock,omitempty"`
	CancunBlock         *big.Int `json:"cancunBlock,omitempty"`
}

// Rules represents the active forks for a specific block number.
type Rules struct {
	ChainID          *big.Int
	IsHomestead      bool
	IsByzantium      bool
	IsConstantinople bool
	IsIstanbul       bool
	IsLondon         bool
	IsParis          bool
	IsShanghai       bool
	IsCancun         bool
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
		IsByzantium:      isForked(c.ByzantiumBlock, num),
		IsConstantinople: isForked(c.ConstantinopleBlock, num),
		IsIstanbul:       isForked(c.IstanbulBlock, num),
		IsLondon:         isForked(c.LondonBlock, num),
		IsParis:          isForked(c.ParisBlock, num),
		IsShanghai:       isForked(c.ShanghaiBlock, num),
		IsCancun:         isForked(c.CancunBlock, num),
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
	ByzantiumBlock:      big.NewInt(0),
	ConstantinopleBlock: big.NewInt(0),
	IstanbulBlock:       big.NewInt(0),
	LondonBlock:         big.NewInt(0),
	ParisBlock:          big.NewInt(0),
	ShanghaiBlock:       big.NewInt(0),
	CancunBlock:         big.NewInt(0),
}
