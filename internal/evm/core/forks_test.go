package core

import (
	"math/big"
	"testing"
)

func TestChainConfig_Rules(t *testing.T) {
	config := &ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(10),
		ByzantiumBlock:      big.NewInt(20),
		ConstantinopleBlock: big.NewInt(30),
		IstanbulBlock:       big.NewInt(40),
		LondonBlock:         big.NewInt(50),
		ParisBlock:          big.NewInt(60), // Merge
		ShanghaiBlock:       big.NewInt(70),
		CancunBlock:         big.NewInt(80),
	}

	tests := []struct {
		blockNum *big.Int
		check    func(Rules) bool
		desc     string
	}{
		{
			big.NewInt(0),
			func(r Rules) bool { return !r.IsHomestead && !r.IsByzantium && !r.IsParis },
			"Genesis",
		},
		{
			big.NewInt(10),
			func(r Rules) bool { return r.IsHomestead && !r.IsByzantium },
			"Homestead",
		},
		{
			big.NewInt(25),
			func(r Rules) bool { return r.IsByzantium && !r.IsConstantinople },
			"Byzantium",
		},
		{
			big.NewInt(60),
			func(r Rules) bool { return r.IsParis && !r.IsShanghai },
			"Paris (Merge)",
		},
		{
			big.NewInt(100),
			func(r Rules) bool { return r.IsCancun },
			"Cancun",
		},
	}

	for _, tt := range tests {
		rules := config.Rules(tt.blockNum)
		if !tt.check(rules) {
			t.Errorf("Rules check failed for block %s (%s)", tt.blockNum, tt.desc)
		}
	}
}

func TestDefaultChainConfig(t *testing.T) {
	config := DefaultChainConfig
	rules := config.Rules(big.NewInt(0))

	if !rules.IsCancun {
		t.Error("DefaultChainConfig should have Cancun active at block 0")
	}
}
