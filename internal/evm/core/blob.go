package core

import "math/big"

const BlobBaseFeeUpdateFraction = 3_338_477

func CalcBlobFee(excess uint64) *big.Int {
	return CalcBlobFeeWithFraction(excess, BlobBaseFeeUpdateFraction)
}

// CalcBlobFeeWithFraction implements the fake exponential used by EIP-4844.
func CalcBlobFeeWithFraction(excess, fraction uint64) *big.Int {
	numerator := new(big.Int).SetUint64(excess)
	denominator := new(big.Int).SetUint64(fraction)
	output := new(big.Int)
	accumulator := new(big.Int).Set(denominator)
	for i := int64(1); accumulator.Sign() > 0; i++ {
		output.Add(output, accumulator)
		accumulator.Mul(accumulator, numerator)
		accumulator.Div(accumulator, denominator)
		accumulator.Div(accumulator, big.NewInt(i))
	}
	return output.Div(output, denominator)
}
