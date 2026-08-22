package vm

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"math/big"
	"math/bits"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/smallyunet/echoevm/internal/eth/common"
	"github.com/smallyunet/echoevm/internal/eth/crypto"
	"github.com/smallyunet/echoevm/internal/evm/core"
	"golang.org/x/crypto/ripemd160" //nolint:staticcheck
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
	PrecompileECRecover = common.BytesToAddress([]byte{0x01})
	PrecompileSHA256    = common.BytesToAddress([]byte{0x02})
	PrecompileRIPEMD160 = common.BytesToAddress([]byte{0x03})
	PrecompileIdentity  = common.BytesToAddress([]byte{0x04})
	PrecompileModExp    = common.BytesToAddress([]byte{0x05})
	PrecompileBN256Add  = common.BytesToAddress([]byte{0x06})
	PrecompileBN256Mul  = common.BytesToAddress([]byte{0x07})
	PrecompileBN256Pair = common.BytesToAddress([]byte{0x08})
	PrecompileBlake2F   = common.BytesToAddress([]byte{0x09})
	PrecompileKZG       = common.BytesToAddress([]byte{0x0a})
	PrecompileBLSG1Add  = common.BytesToAddress([]byte{0x0b})
	PrecompileBLSG1MSM  = common.BytesToAddress([]byte{0x0c})
	PrecompileBLSG2Add  = common.BytesToAddress([]byte{0x0d})
	PrecompileBLSG2MSM  = common.BytesToAddress([]byte{0x0e})
	PrecompileBLSPair   = common.BytesToAddress([]byte{0x0f})
	PrecompileBLSMapG1  = common.BytesToAddress([]byte{0x10})
	PrecompileBLSMapG2  = common.BytesToAddress([]byte{0x11})
	PrecompileP256      = common.BytesToAddress([]byte{0x01, 0x00})
)

// PrecompiledContracts maps addresses to their precompiled contract implementations.
var PrecompiledContracts = map[common.Address]PrecompiledContract{
	PrecompileECRecover: &ecrecover{},
	PrecompileSHA256:    &sha256hash{},
	PrecompileRIPEMD160: &ripemd160hash{},
	PrecompileIdentity:  &dataCopy{},
	PrecompileModExp:    &modExp{},
	PrecompileBN256Add:  &bn256Add{gas: 150},
	PrecompileBN256Mul:  &bn256ScalarMul{gas: 6000},
	PrecompileBN256Pair: &bn256Pairing{baseGas: 45000, perPairGas: 34000},
	PrecompileBlake2F:   &blake2F{},
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

// precompiledContractsForRules selects EchoEVM-owned consensus precompiles for
// the active fork. No execution-client implementation participates here.
func precompiledContractsForRules(rules core.Rules) map[common.Address]PrecompiledContract {
	contracts := map[common.Address]PrecompiledContract{
		PrecompileECRecover: &ecrecover{}, PrecompileSHA256: &sha256hash{},
		PrecompileRIPEMD160: &ripemd160hash{}, PrecompileIdentity: &dataCopy{},
	}
	if rules.IsByzantium {
		contracts[PrecompileModExp] = &modExp{minimumGas: 200}
		contracts[PrecompileBN256Add] = &bn256Add{gas: 500}
		contracts[PrecompileBN256Mul] = &bn256ScalarMul{gas: 40000}
		contracts[PrecompileBN256Pair] = &bn256Pairing{baseGas: 100000, perPairGas: 80000}
	}
	if rules.IsIstanbul {
		contracts[PrecompileBN256Add] = &bn256Add{gas: 150}
		contracts[PrecompileBN256Mul] = &bn256ScalarMul{gas: 6000}
		contracts[PrecompileBN256Pair] = &bn256Pairing{baseGas: 45000, perPairGas: 34000}
		contracts[PrecompileBlake2F] = &blake2F{}
	}
	if rules.IsCancun {
		contracts[PrecompileKZG] = &kzgPointEvaluation{}
	}
	if rules.IsPrague {
		contracts[PrecompileBLSG1Add] = &blsG1Add{}
		contracts[PrecompileBLSG1MSM] = &blsG1MSM{}
		contracts[PrecompileBLSG2Add] = &blsG2Add{}
		contracts[PrecompileBLSG2MSM] = &blsG2MSM{}
		contracts[PrecompileBLSPair] = &blsPairing{}
		contracts[PrecompileBLSMapG1] = &blsMapG1{}
		contracts[PrecompileBLSMapG2] = &blsMapG2{}
	}
	if rules.IsOsaka {
		contracts[PrecompileModExp] = &modExp{minimumGas: 500}
		contracts[PrecompileP256] = &p256Verify{}
	}
	return contracts
}

func IsPrecompiledForRules(addr common.Address, rules core.Rules) bool {
	_, ok := precompiledContractsForRules(rules)[addr]
	return ok
}

func RunPrecompiledForRules(addr common.Address, input []byte, suppliedGas uint64, rules core.Rules) ([]byte, uint64, error) {
	p, ok := precompiledContractsForRules(rules)[addr]
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

func ActivePrecompilesForRules(rules core.Rules) []common.Address {
	contracts := precompiledContractsForRules(rules)
	addresses := make([]common.Address, 0, len(contracts))
	for address := range contracts {
		addresses = append(addresses, address)
	}
	return addresses
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

// =============================================================================
// MODEXP (0x05) - Modular Exponentiation
// =============================================================================

type modExp struct{ minimumGas uint64 }

func (c *modExp) RequiredGas(input []byte) uint64 {
	padded := common.RightPadBytes(input, 96)
	baseLen := new(big.Int).SetBytes(padded[:32])
	expLen := new(big.Int).SetBytes(padded[32:64])
	modLen := new(big.Int).SetBytes(padded[64:96])
	if !baseLen.IsUint64() || !expLen.IsUint64() || !modLen.IsUint64() {
		return math.MaxUint64
	}
	bLen, eLen, mLen := baseLen.Uint64(), expLen.Uint64(), modLen.Uint64()
	maxLen := max(bLen, mLen)
	words, carry := bits.Add64(maxLen, 7, 0)
	if carry != 0 {
		return math.MaxUint64
	}
	words /= 8
	carry, complexity := bits.Mul64(words, words)
	if carry != 0 {
		return math.MaxUint64
	}
	osaka := c.minimumGas >= 500
	if osaka {
		if maxLen <= 32 {
			complexity = 16
		} else {
			carry, complexity = bits.Mul64(complexity, 2)
			if carry != 0 {
				return math.MaxUint64
			}
		}
	}
	headLen := min(eLen, 32)
	head := make([]byte, headLen)
	if headLen > 0 {
		start := uint64(96) + bLen
		if start < uint64(len(input)) {
			copy(head, input[start:min(start+headLen, uint64(len(input)))])
		}
	}
	iteration := uint64(0)
	multiplier := uint64(8)
	if osaka {
		multiplier = 16
	}
	if eLen > 32 {
		carry, iteration = bits.Mul64(eLen-32, multiplier)
		if carry != 0 {
			return math.MaxUint64
		}
	}
	if exponent := new(big.Int).SetBytes(head); exponent.Sign() > 0 {
		iteration += uint64(exponent.BitLen() - 1)
	}
	iteration = max(iteration, 1)
	if complexity != 0 && iteration > math.MaxUint64/complexity {
		return math.MaxUint64
	}
	gas := complexity * iteration
	if !osaka {
		gas /= 3
	}
	minimum := c.minimumGas
	if minimum == 0 {
		minimum = 200
	}
	return max(gas, minimum)
}

func (c *modExp) Run(input []byte) ([]byte, error) {
	// Layout:
	// Length of Base (32 bytes)
	// Length of Exponent (32 bytes)
	// Length of Modulus (32 bytes)
	// Base (B bytes)
	// Exponent (E bytes)
	// Modulus (M bytes)

	var baseLen, expLen, modLen *big.Int

	// Ensure input has at least valid length headers roughly
	// But strictly, we should just read as much as available
	padded := common.RightPadBytes(input, 96)
	baseLen = new(big.Int).SetBytes(padded[0:32])
	expLen = new(big.Int).SetBytes(padded[32:64])
	modLen = new(big.Int).SetBytes(padded[64:96])

	bLen := baseLen.Uint64()
	eLen := expLen.Uint64()
	mLen := modLen.Uint64()

	// EIP-7823 caps every operand length at 1024 bytes in Osaka.
	if c.minimumGas >= 500 && (baseLen.BitLen() > 64 || expLen.BitLen() > 64 || modLen.BitLen() > 64 || bLen > 1024 || eLen > 1024 || mLen > 1024) {
		return nil, errors.New("modexp: operand length exceeds Osaka 1024-byte limit")
	}
	if bLen > 1024*1024 || eLen > 1024*1024 || mLen > 1024*1024 {
		return nil, errors.New("modexp: input too large")
	}

	start := uint64(96)
	endBase := start + bLen
	endExp := endBase + eLen
	endMod := endExp + mLen

	getData := func(start, end uint64) []byte {
		if start >= uint64(len(input)) {
			return make([]byte, end-start)
		}
		if end > uint64(len(input)) {
			res := make([]byte, end-start)
			copy(res, input[start:])
			return res
		}
		return input[start:end]
	}

	base := new(big.Int).SetBytes(getData(start, endBase))
	exp := new(big.Int).SetBytes(getData(endBase, endExp))
	mod := new(big.Int).SetBytes(getData(endExp, endMod))

	if mod.Sign() == 0 {
		return make([]byte, mLen), nil
	}

	res := new(big.Int).Exp(base, exp, mod)
	return common.LeftPadBytes(res.Bytes(), int(mLen)), nil
}

// =============================================================================
// BN256ADD (0x06) - Alt_bn128 Addition
// =============================================================================

type bn256Add struct{ gas uint64 }

func (c *bn256Add) RequiredGas(input []byte) uint64 {
	return c.gas
}

func (c *bn256Add) Run(input []byte) ([]byte, error) {
	input = common.RightPadBytes(input, 128)

	p1 := new(bn254.G1Affine)
	p2 := new(bn254.G1Affine)

	if _, err := p1.SetBytes(input[0:64]); err != nil {
		return nil, errors.New("bn256Add: invalid first point")
	}
	if _, err := p2.SetBytes(input[64:128]); err != nil {
		return nil, errors.New("bn256Add: invalid second point")
	}

	res := new(bn254.G1Affine)
	res.Add(p1, p2)
	encoded := res.RawBytes()
	return encoded[:], nil
}

// =============================================================================
// BN256MUL (0x07) - Alt_bn128 Scalar Multiplication
// =============================================================================

type bn256ScalarMul struct{ gas uint64 }

func (c *bn256ScalarMul) RequiredGas(input []byte) uint64 {
	return c.gas
}

func (c *bn256ScalarMul) Run(input []byte) ([]byte, error) {
	input = common.RightPadBytes(input, 96)

	p := new(bn254.G1Affine)
	if _, err := p.SetBytes(input[0:64]); err != nil {
		return nil, errors.New("bn256ScalarMul: invalid point")
	}

	scalar := new(big.Int).SetBytes(input[64:96])

	res := new(bn254.G1Affine)
	res.ScalarMultiplication(p, scalar)
	encoded := res.RawBytes()
	return encoded[:], nil
}

// =============================================================================
// BN256PAIRING (0x08) - Alt_bn128 Pairing Check
// =============================================================================

type bn256Pairing struct {
	baseGas, perPairGas uint64
}

func (c *bn256Pairing) RequiredGas(input []byte) uint64 {
	elementCount := uint64(len(input) / 192)
	return c.baseGas + elementCount*c.perPairGas
}

func (c *bn256Pairing) Run(input []byte) ([]byte, error) {
	// Input is a list of (p1, p2) pairs
	// p1 is G1 (64 bytes), p2 is G2 (128 bytes) => 192 bytes total per pair
	if len(input)%192 != 0 {
		return nil, errors.New("bn256Pairing: invalid input length")
	}

	var points []bn254.G1Affine
	var twisted []bn254.G2Affine

	for i := 0; i < len(input); i += 192 {
		p1 := new(bn254.G1Affine)
		if _, err := p1.SetBytes(input[i : i+64]); err != nil {
			return nil, errors.New("bn256Pairing: invalid G1 point")
		}

		p2 := new(bn254.G2Affine)
		if _, err := p2.SetBytes(input[i+64 : i+192]); err != nil {
			return nil, errors.New("bn256Pairing: invalid G2 point")
		}

		points = append(points, *p1)
		twisted = append(twisted, *p2)
	}

	valid, err := bn254.PairingCheck(points, twisted)
	if err != nil {
		return nil, err
	}
	if valid {
		return common.LeftPadBytes([]byte{1}, 32), nil
	}
	return common.LeftPadBytes([]byte{0}, 32), nil
}

// =============================================================================
// BLAKE2F (0x09) - BLAKE2b Compression Function F
// =============================================================================

type blake2F struct{}

func (c *blake2F) RequiredGas(input []byte) uint64 {
	if len(input) != 213 {
		return 0
	}
	return uint64(binary.BigEndian.Uint32(input[:4]))
}

func (c *blake2F) Run(input []byte) ([]byte, error) {
	if len(input) != 213 {
		return nil, errors.New("blake2f: invalid input length")
	}
	if input[212] != 0 && input[212] != 1 {
		return nil, errors.New("blake2f: invalid final flag")
	}

	rounds := binary.BigEndian.Uint32(input[:4])
	var h [8]uint64
	var m [16]uint64
	var counter [2]uint64
	for index := range h {
		offset := 4 + index*8
		h[index] = binary.LittleEndian.Uint64(input[offset : offset+8])
	}
	for index := range m {
		offset := 68 + index*8
		m[index] = binary.LittleEndian.Uint64(input[offset : offset+8])
	}
	counter[0] = binary.LittleEndian.Uint64(input[196:204])
	counter[1] = binary.LittleEndian.Uint64(input[204:212])

	blake2bCompress(&h, m, counter, input[212] == 1, rounds)
	output := make([]byte, 64)
	for index, word := range h {
		binary.LittleEndian.PutUint64(output[index*8:(index+1)*8], word)
	}
	return output, nil
}
