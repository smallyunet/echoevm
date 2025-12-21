package vm

import (
	"crypto/sha256"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/ripemd160"
)

// PrecompiledContract is the interface for native precompiled contract implementations.
type PrecompiledContract interface {
	// RequiredGas calculates the gas cost for running the precompile with the given input.
	RequiredGas(input []byte) uint64
	// Run executes the precompiled contract and returns the output or an error.
	Run(input []byte) ([]byte, error)
}

// Precompiled contract addresses
var (
	PrecompileECRecover  = common.BytesToAddress([]byte{0x01})
	PrecompileSHA256     = common.BytesToAddress([]byte{0x02})
	PrecompileRIPEMD160  = common.BytesToAddress([]byte{0x03})
	PrecompileIdentity   = common.BytesToAddress([]byte{0x04})
)

// PrecompiledContracts maps addresses to their precompiled contract implementations.
var PrecompiledContracts = map[common.Address]PrecompiledContract{
	PrecompileECRecover:  &ecrecover{},
	PrecompileSHA256:     &sha256hash{},
	PrecompileRIPEMD160:  &ripemd160hash{},
	PrecompileIdentity:   &dataCopy{},
}

// IsPrecompiled returns true if the address is a precompiled contract.
func IsPrecompiled(addr common.Address) bool {
	_, ok := PrecompiledContracts[addr]
	return ok
}

// RunPrecompiled executes a precompiled contract and returns the output and remaining gas.
func RunPrecompiled(addr common.Address, input []byte, suppliedGas uint64) ([]byte, uint64, error) {
	p, ok := PrecompiledContracts[addr]
	if !ok {
		return nil, suppliedGas, errors.New("precompiled contract not found")
	}

	gasCost := p.RequiredGas(input)
	if suppliedGas < gasCost {
		return nil, 0, errors.New("out of gas")
	}

	output, err := p.Run(input)
	return output, suppliedGas - gasCost, err
}

// =============================================================================
// ECRECOVER (0x01) - Elliptic curve public key recovery
// =============================================================================

type ecrecover struct{}

func (c *ecrecover) RequiredGas(input []byte) uint64 {
	return 3000 // Fixed gas cost
}

func (c *ecrecover) Run(input []byte) ([]byte, error) {
	const ecRecoverInputLength = 128

	// Pad input to expected length
	input = common.RightPadBytes(input, ecRecoverInputLength)

	// Extract components: hash (32) + v (32) + r (32) + s (32)
	hash := input[0:32]
	v := new(big.Int).SetBytes(input[32:64])
	r := new(big.Int).SetBytes(input[64:96])
	s := new(big.Int).SetBytes(input[96:128])

	// Validate v: must be 27 or 28
	if !allZero(input[32:63]) || !isValidV(v) {
		return nil, nil // Invalid input returns empty, not error
	}

	// Validate r and s: must be in valid range (> 0 and < secp256k1n)
	secp256k1N, _ := new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	if r.Sign() <= 0 || r.Cmp(secp256k1N) >= 0 {
		return nil, nil
	}
	if s.Sign() <= 0 || s.Cmp(secp256k1N) >= 0 {
		return nil, nil
	}

	// Construct signature (r || s || v-27)
	sig := make([]byte, 65)
	r.FillBytes(sig[0:32])
	s.FillBytes(sig[32:64])
	sig[64] = byte(v.Uint64() - 27)

	// Recover public key
	pubKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return nil, nil // Recovery failed, return empty
	}

	// Convert to address and return left-padded to 32 bytes
	addr := crypto.PubkeyToAddress(*pubKey)
	return common.LeftPadBytes(addr.Bytes(), 32), nil
}

func isValidV(v *big.Int) bool {
	return v.Cmp(big.NewInt(27)) == 0 || v.Cmp(big.NewInt(28)) == 0
}

func allZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

// =============================================================================
// SHA256 (0x02) - SHA-256 hash function
// =============================================================================

type sha256hash struct{}

func (c *sha256hash) RequiredGas(input []byte) uint64 {
	// 60 base + 12 per word (32 bytes)
	words := uint64((len(input) + 31) / 32)
	return 60 + 12*words
}

func (c *sha256hash) Run(input []byte) ([]byte, error) {
	h := sha256.Sum256(input)
	return h[:], nil
}

// =============================================================================
// RIPEMD160 (0x03) - RIPEMD-160 hash function
// =============================================================================

type ripemd160hash struct{}

func (c *ripemd160hash) RequiredGas(input []byte) uint64 {
	// 600 base + 120 per word (32 bytes)
	words := uint64((len(input) + 31) / 32)
	return 600 + 120*words
}

func (c *ripemd160hash) Run(input []byte) ([]byte, error) {
	ripemd := ripemd160.New()
	ripemd.Write(input)
	// RIPEMD160 returns 20 bytes, left-pad to 32 bytes
	return common.LeftPadBytes(ripemd.Sum(nil), 32), nil
}

// =============================================================================
// IDENTITY (0x04) - Data copy / identity function
// =============================================================================

type dataCopy struct{}

func (c *dataCopy) RequiredGas(input []byte) uint64 {
	// 15 base + 3 per word (32 bytes)
	words := uint64((len(input) + 31) / 32)
	return 15 + 3*words
}

func (c *dataCopy) Run(input []byte) ([]byte, error) {
	// Simply return a copy of the input
	output := make([]byte, len(input))
	copy(output, input)
	return output, nil
}
