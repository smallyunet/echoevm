package vm

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/consensys/gnark-crypto/ecc"
	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	goethkzg "github.com/crate-crypto/go-eth-kzg"
)

var (
	errModernInvalidLength = errors.New("invalid precompile input length")
	true32                 = append(make([]byte, 31), 1)
)

// kzgPointEvaluation implements EIP-4844 without relying on an execution client.
type kzgPointEvaluation struct{}

func (*kzgPointEvaluation) RequiredGas([]byte) uint64 { return 50000 }

var (
	kzgOnce    sync.Once
	kzgContext *goethkzg.Context
	kzgErr     error
)

func (*kzgPointEvaluation) Run(input []byte) ([]byte, error) {
	if len(input) != 192 {
		return nil, errModernInvalidLength
	}
	var commitment goethkzg.KZGCommitment
	copy(commitment[:], input[96:144])
	versioned := sha256.Sum256(commitment[:])
	versioned[0] = 1
	if !equalBytes(versioned[:], input[:32]) {
		return nil, errors.New("mismatched KZG versioned hash")
	}
	var point, claim goethkzg.Scalar
	var proof goethkzg.KZGProof
	copy(point[:], input[32:64])
	copy(claim[:], input[64:96])
	copy(proof[:], input[144:192])
	kzgOnce.Do(func() { kzgContext, kzgErr = goethkzg.NewContext4096Secure() })
	if kzgErr != nil {
		return nil, fmt.Errorf("initialize KZG context: %w", kzgErr)
	}
	if err := kzgContext.VerifyKZGProof(commitment, point, claim, proof); err != nil {
		return nil, fmt.Errorf("verify KZG proof: %w", err)
	}
	out := make([]byte, 64)
	out[30], out[31] = 0x10, 0x00 // FIELD_ELEMENTS_PER_BLOB = 4096
	copy(out[32:], goethkzg.BlsModulus[:])
	return out, nil
}

// p256Verify implements EIP-7951.
type p256Verify struct{}

func (*p256Verify) RequiredGas([]byte) uint64 { return 6900 }
func (*p256Verify) Run(input []byte) ([]byte, error) {
	if len(input) != 160 {
		return nil, nil
	}
	r := new(big.Int).SetBytes(input[32:64])
	s := new(big.Int).SetBytes(input[64:96])
	x := new(big.Int).SetBytes(input[96:128])
	y := new(big.Int).SetBytes(input[128:160])
	curve := elliptic.P256()
	encoded := append([]byte{0x04}, input[96:160]...)
	validatedX, validatedY := elliptic.Unmarshal(curve, encoded)
	if validatedX == nil || validatedX.Cmp(x) != 0 || validatedY.Cmp(y) != 0 || !ecdsa.Verify(&ecdsa.PublicKey{Curve: curve, X: validatedX, Y: validatedY}, input[:32], r, s) {
		return nil, nil
	}
	return append([]byte(nil), true32...), nil
}

var (
	errBLSFieldTopBytes = errors.New("BLS12-381 field element has non-zero top bytes")
	errBLSG1Subgroup    = errors.New("BLS12-381 G1 point is not in subgroup")
	errBLSG2Subgroup    = errors.New("BLS12-381 G2 point is not in subgroup")
)

func decodeBLSField(in []byte) (fp.Element, error) {
	if len(in) != 64 {
		return fp.Element{}, errModernInvalidLength
	}
	for _, b := range in[:16] {
		if b != 0 {
			return fp.Element{}, errBLSFieldTopBytes
		}
	}
	var raw [48]byte
	copy(raw[:], in[16:])
	return fp.BigEndian.Element(&raw)
}

func decodeBLSG1(in []byte) (*bls12381.G1Affine, error) {
	if len(in) != 128 {
		return nil, errModernInvalidLength
	}
	x, err := decodeBLSField(in[:64])
	if err != nil {
		return nil, err
	}
	y, err := decodeBLSField(in[64:])
	if err != nil {
		return nil, err
	}
	p := &bls12381.G1Affine{X: x, Y: y}
	if !p.IsOnCurve() {
		return nil, errors.New("BLS12-381 G1 point is not on curve")
	}
	return p, nil
}

func decodeBLSG2(in []byte) (*bls12381.G2Affine, error) {
	if len(in) != 256 {
		return nil, errModernInvalidLength
	}
	x0, err := decodeBLSField(in[:64])
	if err != nil {
		return nil, err
	}
	x1, err := decodeBLSField(in[64:128])
	if err != nil {
		return nil, err
	}
	y0, err := decodeBLSField(in[128:192])
	if err != nil {
		return nil, err
	}
	y1, err := decodeBLSField(in[192:])
	if err != nil {
		return nil, err
	}
	p := &bls12381.G2Affine{X: bls12381.E2{A0: x0, A1: x1}, Y: bls12381.E2{A0: y0, A1: y1}}
	if !p.IsOnCurve() {
		return nil, errors.New("BLS12-381 G2 point is not on curve")
	}
	return p, nil
}

func encodeBLSG1(p *bls12381.G1Affine) []byte {
	out := make([]byte, 128)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[16:64]), p.X)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[80:128]), p.Y)
	return out
}

func encodeBLSG2(p *bls12381.G2Affine) []byte {
	out := make([]byte, 256)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[16:64]), p.X.A0)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[80:128]), p.X.A1)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[144:192]), p.Y.A0)
	fp.BigEndian.PutElement((*[fp.Bytes]byte)(out[208:256]), p.Y.A1)
	return out
}

type blsG1Add struct{}

func (*blsG1Add) RequiredGas([]byte) uint64 { return 375 }
func (*blsG1Add) Run(input []byte) ([]byte, error) {
	if len(input) != 256 {
		return nil, errModernInvalidLength
	}
	a, err := decodeBLSG1(input[:128])
	if err != nil {
		return nil, err
	}
	b, err := decodeBLSG1(input[128:])
	if err != nil {
		return nil, err
	}
	return encodeBLSG1(new(bls12381.G1Affine).Add(a, b)), nil
}

type blsG1MSM struct{}

func (*blsG1MSM) RequiredGas(input []byte) uint64 {
	return blsMSMGas(len(input)/160, 12000, blsG1Discount[:])
}
func (*blsG1MSM) Run(input []byte) ([]byte, error) {
	if len(input) == 0 || len(input)%160 != 0 {
		return nil, errModernInvalidLength
	}
	k := len(input) / 160
	points := make([]bls12381.G1Affine, k)
	scalars := make([]fr.Element, k)
	for i := range k {
		p, err := decodeBLSG1(input[i*160 : i*160+128])
		if err != nil {
			return nil, err
		}
		if !p.IsInSubGroup() {
			return nil, errBLSG1Subgroup
		}
		points[i] = *p
		scalars[i].SetBytes(input[i*160+128 : i*160+160])
	}
	r, err := new(bls12381.G1Affine).MultiExp(points, scalars, ecc.MultiExpConfig{})
	if err != nil {
		return nil, err
	}
	return encodeBLSG1(r), nil
}

type blsG2Add struct{}

func (*blsG2Add) RequiredGas([]byte) uint64 { return 600 }
func (*blsG2Add) Run(input []byte) ([]byte, error) {
	if len(input) != 512 {
		return nil, errModernInvalidLength
	}
	a, err := decodeBLSG2(input[:256])
	if err != nil {
		return nil, err
	}
	b, err := decodeBLSG2(input[256:])
	if err != nil {
		return nil, err
	}
	return encodeBLSG2(new(bls12381.G2Affine).Add(a, b)), nil
}

type blsG2MSM struct{}

func (*blsG2MSM) RequiredGas(input []byte) uint64 {
	return blsMSMGas(len(input)/288, 22500, blsG2Discount[:])
}
func (*blsG2MSM) Run(input []byte) ([]byte, error) {
	if len(input) == 0 || len(input)%288 != 0 {
		return nil, errModernInvalidLength
	}
	k := len(input) / 288
	points := make([]bls12381.G2Affine, k)
	scalars := make([]fr.Element, k)
	for i := range k {
		p, err := decodeBLSG2(input[i*288 : i*288+256])
		if err != nil {
			return nil, err
		}
		if !p.IsInSubGroup() {
			return nil, errBLSG2Subgroup
		}
		points[i] = *p
		scalars[i].SetBytes(input[i*288+256 : i*288+288])
	}
	r, err := new(bls12381.G2Affine).MultiExp(points, scalars, ecc.MultiExpConfig{})
	if err != nil {
		return nil, err
	}
	return encodeBLSG2(r), nil
}

type blsPairing struct{}

func (*blsPairing) RequiredGas(input []byte) uint64 { return 37700 + uint64(len(input)/384)*32600 }
func (*blsPairing) Run(input []byte) ([]byte, error) {
	if len(input) == 0 || len(input)%384 != 0 {
		return nil, errModernInvalidLength
	}
	count := len(input) / 384
	g1 := make([]bls12381.G1Affine, count)
	g2 := make([]bls12381.G2Affine, count)
	for i := range count {
		p, err := decodeBLSG1(input[i*384 : i*384+128])
		if err != nil {
			return nil, err
		}
		q, err := decodeBLSG2(input[i*384+128 : i*384+384])
		if err != nil {
			return nil, err
		}
		if !p.IsInSubGroup() {
			return nil, errBLSG1Subgroup
		}
		if !q.IsInSubGroup() {
			return nil, errBLSG2Subgroup
		}
		g1[i], g2[i] = *p, *q
	}
	ok, err := bls12381.PairingCheck(g1, g2)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 32)
	if ok {
		out[31] = 1
	}
	return out, nil
}

type blsMapG1 struct{}

func (*blsMapG1) RequiredGas([]byte) uint64 { return 5500 }
func (*blsMapG1) Run(input []byte) ([]byte, error) {
	if len(input) != 64 {
		return nil, errModernInvalidLength
	}
	e, err := decodeBLSField(input)
	if err != nil {
		return nil, err
	}
	r := bls12381.MapToG1(e)
	return encodeBLSG1(&r), nil
}

type blsMapG2 struct{}

func (*blsMapG2) RequiredGas([]byte) uint64 { return 23800 }
func (*blsMapG2) Run(input []byte) ([]byte, error) {
	if len(input) != 128 {
		return nil, errModernInvalidLength
	}
	c0, err := decodeBLSField(input[:64])
	if err != nil {
		return nil, err
	}
	c1, err := decodeBLSField(input[64:])
	if err != nil {
		return nil, err
	}
	r := bls12381.MapToG2(bls12381.E2{A0: c0, A1: c1})
	return encodeBLSG2(&r), nil
}

func blsMSMGas(k int, multiplicationGas uint64, discount []uint64) uint64 {
	if k == 0 {
		return 0
	}
	i := k - 1
	if i >= len(discount) {
		i = len(discount) - 1
	}
	return uint64(k) * multiplicationGas * discount[i] / 1000
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var different byte
	for i := range a {
		different |= a[i] ^ b[i]
	}
	return different == 0
}

var blsG1Discount = [...]uint64{1000, 949, 848, 797, 764, 750, 738, 728, 719, 712, 705, 698, 692, 687, 682, 677, 673, 669, 665, 661, 658, 654, 651, 648, 645, 642, 640, 637, 635, 632, 630, 627, 625, 623, 621, 619, 617, 615, 613, 611, 609, 608, 606, 604, 603, 601, 599, 598, 596, 595, 593, 592, 591, 589, 588, 586, 585, 584, 582, 581, 580, 579, 577, 576, 575, 574, 573, 572, 570, 569, 568, 567, 566, 565, 564, 563, 562, 561, 560, 559, 558, 557, 556, 555, 554, 553, 552, 551, 550, 549, 548, 547, 547, 546, 545, 544, 543, 542, 541, 540, 540, 539, 538, 537, 536, 536, 535, 534, 533, 532, 532, 531, 530, 529, 528, 528, 527, 526, 525, 525, 524, 523, 522, 522, 521, 520, 520, 519}
var blsG2Discount = [...]uint64{1000, 1000, 923, 884, 855, 832, 812, 796, 782, 770, 759, 749, 740, 732, 724, 717, 711, 704, 699, 693, 688, 683, 679, 674, 670, 666, 663, 659, 655, 652, 649, 646, 643, 640, 637, 634, 632, 629, 627, 624, 622, 620, 618, 615, 613, 611, 609, 607, 606, 604, 602, 600, 598, 597, 595, 593, 592, 590, 589, 587, 586, 584, 583, 582, 580, 579, 578, 576, 575, 574, 573, 571, 570, 569, 568, 567, 566, 565, 563, 562, 561, 560, 559, 558, 557, 556, 555, 554, 553, 552, 552, 551, 550, 549, 548, 547, 546, 545, 545, 544, 543, 542, 541, 541, 540, 539, 538, 537, 537, 536, 535, 535, 534, 533, 532, 532, 531, 530, 530, 529, 528, 528, 527, 526, 526, 525, 524, 524}
