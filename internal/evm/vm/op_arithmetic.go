// op_arithmetic.go
package vm

import "math/big"

func opAdd(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	res := new(big.Int).Add(x, y)
	res.And(res, mask256)
	i.stack.PushSafe(res)
}

func opSub(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	// EVM uses two's complement for subtraction, but big.Int handles it.
	// We just need to mask the result to 256 bits to get the correct wrap-around behavior.
	res := new(big.Int).Sub(x, y)
	res.And(res, mask256)
	i.stack.PushSafe(res)
}

func opMul(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	res := new(big.Int).Mul(x, y)
	res.And(res, mask256)
	i.stack.PushSafe(res)
}

func opExp(i *Interpreter, _ byte) {
	base := i.stack.PopSafe()
	exp := i.stack.PopSafe()
	r := new(big.Int).Exp(base, exp, twoTo256)
	i.stack.PushSafe(r)
}

func opDiv(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	if y.Sign() == 0 {
		i.stack.PushSafe(big.NewInt(0))
	} else {
		i.stack.PushSafe(new(big.Int).Div(x, y))
	}
}

// opSdiv performs signed integer division
func opSdiv(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	if y.Sign() == 0 {
		i.stack.PushSafe(big.NewInt(0))
		return
	}
	// Convert to signed
	sx := toSigned(x)
	sy := toSigned(y)

	// Handle special case: -2^255 / -1 = -2^255 (overflow)
	if sx.Cmp(new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 255))) == 0 && sy.Cmp(big.NewInt(-1)) == 0 {
		i.stack.PushSafe(new(big.Int).Lsh(big.NewInt(1), 255))
		return
	}

	res := new(big.Int).Quo(sx, sy) // Quo truncates toward zero
	res.And(res, mask256)
	i.stack.PushSafe(res)
}

func opMod(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	if y.Sign() == 0 {
		i.stack.PushSafe(big.NewInt(0))
	} else {
		i.stack.PushSafe(new(big.Int).Mod(x, y))
	}
}

// opSmod performs signed integer modulo
func opSmod(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	if y.Sign() == 0 {
		i.stack.PushSafe(big.NewInt(0))
		return
	}
	// Convert to signed
	sx := toSigned(x)
	sy := toSigned(y)

	res := new(big.Int).Rem(sx, sy) // Rem has the sign of the dividend
	res.And(res, mask256)
	i.stack.PushSafe(res)
}

// toSigned converts a 256-bit unsigned value to a signed big.Int
func toSigned(val *big.Int) *big.Int {
	result := new(big.Int).Set(val)
	if val.Bit(255) == 1 {
		result.Sub(result, twoTo256)
	}
	return result
}

// opAddmod pops (a, b, m) and pushes (a + b) % m. If m == 0 pushes 0.
// Values are treated as 256-bit unsigned integers.
func opAddmod(i *Interpreter, _ byte) {
	m := i.stack.PopSafe()
	b := i.stack.PopSafe()
	a := i.stack.PopSafe()
	if m.Sign() == 0 { // modulus zero yields 0
		i.stack.PushSafe(big.NewInt(0))
		return
	}
	// (a + b) % m with 256-bit wrap before mod (though Go big.Int handles big values already, we mask to 256 bits explicitly)
	sum := new(big.Int).Add(a, b)
	sum.And(sum, mask256)
	sum.Mod(sum, m)
	i.stack.PushSafe(sum)
}

// opMulmod pops (a, b, m) and pushes (a * b) % m. If m == 0 pushes 0.
// Uses 512-bit intermediate reduced back by modulo, then masked to 256 bits.
func opMulmod(i *Interpreter, _ byte) {
	m := i.stack.PopSafe()
	b := i.stack.PopSafe()
	a := i.stack.PopSafe()
	if m.Sign() == 0 {
		i.stack.PushSafe(big.NewInt(0))
		return
	}
	prod := new(big.Int).Mul(a, b)
	// EVM semantics: values are 256-bit, though multiplication can exceed; modulus then masked.
	prod.Mod(prod, m)
	prod.And(prod, mask256)
	i.stack.PushSafe(prod)
}

func opEq(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	if x.Cmp(y) == 0 {
		i.stack.PushSafe(big.NewInt(1))
	} else {
		i.stack.PushSafe(big.NewInt(0))
	}
}

func opIsZero(i *Interpreter, _ byte) {
	x := i.stack.PopSafe()
	if x.Sign() == 0 {
		i.stack.PushSafe(big.NewInt(1))
	} else {
		i.stack.PushSafe(big.NewInt(0))
	}
}

// opLt compares the top two stack values and pushes 1 if the first
// is strictly less than the second, otherwise pushes 0.
func opLt(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	if x.Cmp(y) < 0 {
		i.stack.PushSafe(big.NewInt(1))
	} else {
		i.stack.PushSafe(big.NewInt(0))
	}
}

// opGt compares the top two stack values and pushes 1 if the first
// is strictly greater than the second.
func opGt(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	if x.Cmp(y) > 0 {
		i.stack.PushSafe(big.NewInt(1))
	} else {
		i.stack.PushSafe(big.NewInt(0))
	}
}

func opSgt(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	sx := new(big.Int).Set(x)
	if x.Bit(255) == 1 {
		sx.Sub(sx, twoTo256)
	}
	sy := new(big.Int).Set(y)
	if y.Bit(255) == 1 {
		sy.Sub(sy, twoTo256)
	}
	if sx.Cmp(sy) > 0 {
		i.stack.PushSafe(big.NewInt(1))
	} else {
		i.stack.PushSafe(big.NewInt(0))
	}
}

// opSlt compares two signed 256-bit integers and pushes 1 if the first is
// strictly less than the second.
func opSlt(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	sx := new(big.Int).Set(x)
	if x.Bit(255) == 1 {
		sx.Sub(sx, twoTo256)
	}
	sy := new(big.Int).Set(y)
	if y.Bit(255) == 1 {
		sy.Sub(sy, twoTo256)
	}
	if sx.Cmp(sy) < 0 {
		i.stack.PushSafe(big.NewInt(1))
	} else {
		i.stack.PushSafe(big.NewInt(0))
	}
}

// opSignextend extends the sign bit of a value from the specified byte
// position.
func opSignextend(i *Interpreter, _ byte) {
	back := i.stack.PopSafe().Uint64()
	val := i.stack.PopSafe()
	if back >= 32 {
		i.stack.PushSafe(val)
		return
	}
	bit := uint(back*8 + 7)
	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), bit+1), big.NewInt(1))
	if val.Bit(int(bit)) == 1 {
		res := new(big.Int).Or(val, new(big.Int).Not(mask))
		res.And(res, mask256)
		i.stack.PushSafe(res)
	} else {
		res := new(big.Int).And(val, mask)
		i.stack.PushSafe(res)
	}
}
