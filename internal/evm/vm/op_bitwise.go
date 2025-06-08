package vm

import "math/big"

var (
	twoTo256 = new(big.Int).Lsh(big.NewInt(1), 256)
	mask256  = new(big.Int).Sub(twoTo256, big.NewInt(1))
)

func opAnd(i *Interpreter, _ byte) {
	x, y := i.stack.Pop(), i.stack.Pop()
	i.stack.Push(new(big.Int).And(x, y))
}

func opOr(i *Interpreter, _ byte) {
	x, y := i.stack.Pop(), i.stack.Pop()
	i.stack.Push(new(big.Int).Or(x, y))
}

func opXor(i *Interpreter, _ byte) {
	x, y := i.stack.Pop(), i.stack.Pop()
	i.stack.Push(new(big.Int).Xor(x, y))
}

func opNot(i *Interpreter, _ byte) {
	x := i.stack.Pop()
	x.Not(x)
	x.And(x, mask256)
	i.stack.Push(x)
}

func opByte(i *Interpreter, _ byte) {
	pos := i.stack.Pop().Uint64()
	word := i.stack.Pop()
	if pos >= 32 {
		i.stack.Push(big.NewInt(0))
		return
	}
	buf := make([]byte, 32)
	b := word.Bytes()
	copy(buf[32-len(b):], b)
	i.stack.Push(new(big.Int).SetUint64(uint64(buf[pos])))
}

func opShl(i *Interpreter, _ byte) {
	shift := i.stack.Pop().Uint64()
	val := i.stack.Pop()
	if shift >= 256 {
		i.stack.Push(big.NewInt(0))
		return
	}
	r := new(big.Int).Lsh(val, uint(shift))
	r.And(r, mask256)
	i.stack.Push(r)
}

func opShr(i *Interpreter, _ byte) {
	shift := i.stack.Pop().Uint64()
	val := i.stack.Pop()
	if shift >= 256 {
		i.stack.Push(big.NewInt(0))
		return
	}
	r := new(big.Int).Rsh(val, uint(shift))
	i.stack.Push(r)
}

func opSar(i *Interpreter, _ byte) {
	shift := i.stack.Pop().Uint64()
	val := i.stack.Pop()
	if shift >= 256 {
		if val.Bit(255) == 1 {
			i.stack.Push(new(big.Int).Set(mask256))
		} else {
			i.stack.Push(big.NewInt(0))
		}
		return
	}
	signed := new(big.Int).Set(val)
	if val.Bit(255) == 1 {
		signed.Sub(signed, twoTo256)
	}
	signed.Rsh(signed, uint(shift))
	if signed.Sign() < 0 {
		signed.Add(signed, twoTo256)
	}
	signed.And(signed, mask256)
	i.stack.Push(signed)
}
