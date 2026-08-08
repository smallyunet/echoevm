package core

import (
	"math/big"
	"strings"
	"testing"
)

func TestChainConfig_Rules(t *testing.T) {
	config := &ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(10),
		EIP150Block:         big.NewInt(15),
		EIP155Block:         big.NewInt(16),
		EIP158Block:         big.NewInt(16),
		ByzantiumBlock:      big.NewInt(20),
		ConstantinopleBlock: big.NewInt(30),
		IstanbulBlock:       big.NewInt(40),
		BerlinBlock:         big.NewInt(45),
		LondonBlock:         big.NewInt(50),
		ParisBlock:          big.NewInt(60), // Merge
		ShanghaiBlock:       big.NewInt(70),
		CancunBlock:         big.NewInt(80),
		PragueBlock:         big.NewInt(90),
		OsakaBlock:          big.NewInt(100),
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

	if !rules.IsOsaka {
		t.Error("DefaultChainConfig should have Osaka active at block 0")
	}
}

func TestChainConfigForEverySupportedFork(t *testing.T) {
	for _, fork := range SupportedForks {
		t.Run(fork, func(t *testing.T) {
			config, err := ChainConfigForFork(fork)
			if err != nil {
				t.Fatal(err)
			}
			normalized, err := NormalizeFork(strings.ToLower(fork))
			if err != nil || normalized != fork {
				t.Fatalf("NormalizeFork = %q, %v; want %q", normalized, err, fork)
			}
			if fork == ForkFrontier && config.Rules(new(big.Int)).IsHomestead {
				t.Fatal("Frontier config unexpectedly enables Homestead")
			}
			if fork == ForkOsaka && !config.Rules(new(big.Int)).IsOsaka {
				t.Fatal("Osaka config does not enable Osaka")
			}
		})
	}
}
