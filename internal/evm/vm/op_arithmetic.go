// op_arithmetic.go
package vm

import "math/big"

func opAdd(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	i.stack.PushSafe(new(big.Int).Add(x, y))
}

func opSub(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	i.stack.PushSafe(new(big.Int).Sub(x, y))
}

func opMul(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	i.stack.PushSafe(new(big.Int).Mul(x, y))
}

func opExp(i *Interpreter, _ byte) {
	exp := i.stack.PopSafe()
	base := i.stack.PopSafe()
	r := new(big.Int).Exp(base, exp, nil)
	r.And(r, mask256)
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

func opMod(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	if y.Sign() == 0 {
		i.stack.PushSafe(big.NewInt(0))
	} else {
		i.stack.PushSafe(new(big.Int).Mod(x, y))
	}
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
