package vm

import "math/big"

var (
	twoTo256 = new(big.Int).Lsh(big.NewInt(1), 256)
	mask256  = new(big.Int).Sub(twoTo256, big.NewInt(1))
)

func opAnd(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	i.stack.PushSafe(new(big.Int).And(x, y))
}

func opOr(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	i.stack.PushSafe(new(big.Int).Or(x, y))
}

func opXor(i *Interpreter, _ byte) {
	x, y := i.stack.PopSafe(), i.stack.PopSafe()
	i.stack.PushSafe(new(big.Int).Xor(x, y))
}

func opNot(i *Interpreter, _ byte) {
	x := i.stack.PopSafe()
	x.Not(x)
	x.And(x, mask256)
	i.stack.PushSafe(x)
}

func opByte(i *Interpreter, _ byte) {
	pos := i.stack.PopSafe().Uint64()
	word := i.stack.PopSafe()
	if pos >= 32 {
		i.stack.PushSafe(big.NewInt(0))
		return
	}
	buf := make([]byte, 32)
	b := word.Bytes()
	copy(buf[32-len(b):], b)
	i.stack.PushSafe(new(big.Int).SetUint64(uint64(buf[pos])))
}

func opShl(i *Interpreter, _ byte) {
	shift := i.stack.PopSafe()
	val := i.stack.PopSafe()

	// If shift amount is >= 256, result is 0
	if shift.Cmp(big.NewInt(256)) >= 0 {
		i.stack.PushSafe(big.NewInt(0))
		return
	}

	// shift is < 256, so it fits in uint
	r := new(big.Int).Lsh(val, uint(shift.Uint64()))
	r.And(r, mask256)
	i.stack.PushSafe(r)
}

func opShr(i *Interpreter, _ byte) {
	shift := i.stack.PopSafe()
	val := i.stack.PopSafe()

	// If shift amount is >= 256, result is 0
	if shift.Cmp(big.NewInt(256)) >= 0 {
		i.stack.PushSafe(big.NewInt(0))
		return
	}

	r := new(big.Int).Rsh(val, uint(shift.Uint64()))
	i.stack.PushSafe(r)
}

func opSar(i *Interpreter, _ byte) {
	shift := i.stack.PopSafe()
	val := i.stack.PopSafe()

	// If shift amount is >= 256
	if shift.Cmp(big.NewInt(256)) >= 0 {
		if val.Bit(255) == 1 {
			i.stack.PushSafe(new(big.Int).Set(mask256)) // -1 in two's complement
		} else {
			i.stack.PushSafe(big.NewInt(0))
		}
		return
	}

	n := uint(shift.Uint64())
	signed := new(big.Int).Set(val)
	if val.Bit(255) == 1 {
		signed.Sub(signed, twoTo256)
	}
	signed.Rsh(signed, n)
	if signed.Sign() < 0 {
		signed.Add(signed, twoTo256)
	}
	signed.And(signed, mask256)
	i.stack.PushSafe(signed)
}
