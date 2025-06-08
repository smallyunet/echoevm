// op_arithmetic.go
package vm

import "math/big"

func opAdd(i *Interpreter, _ byte) {
	x, y := i.stack.Pop(), i.stack.Pop()
	i.stack.Push(new(big.Int).Add(x, y))
}

func opSub(i *Interpreter, _ byte) {
	x, y := i.stack.Pop(), i.stack.Pop()
	i.stack.Push(new(big.Int).Sub(x, y))
}

func opMul(i *Interpreter, _ byte) {
	x, y := i.stack.Pop(), i.stack.Pop()
	i.stack.Push(new(big.Int).Mul(x, y))
}

func opDiv(i *Interpreter, _ byte) {
	x, y := i.stack.Pop(), i.stack.Pop()
	if y.Sign() == 0 {
		i.stack.Push(big.NewInt(0))
	} else {
		i.stack.Push(new(big.Int).Div(x, y))
	}
}

func opMod(i *Interpreter, _ byte) {
	x, y := i.stack.Pop(), i.stack.Pop()
	if y.Sign() == 0 {
		i.stack.Push(big.NewInt(0))
	} else {
		i.stack.Push(new(big.Int).Mod(x, y))
	}
}

func opEq(i *Interpreter, _ byte) {
	x, y := i.stack.Pop(), i.stack.Pop()
	if x.Cmp(y) == 0 {
		i.stack.Push(big.NewInt(1))
	} else {
		i.stack.Push(big.NewInt(0))
	}
}

func opIsZero(i *Interpreter, _ byte) {
	x := i.stack.Pop()
	if x.Sign() == 0 {
		i.stack.Push(big.NewInt(1))
	} else {
		i.stack.Push(big.NewInt(0))
	}
}

// opLt compares the top two stack values and pushes 1 if the first
// is strictly less than the second, otherwise pushes 0.
func opLt(i *Interpreter, _ byte) {
	x, y := i.stack.Pop(), i.stack.Pop()
	if x.Cmp(y) < 0 {
		i.stack.Push(big.NewInt(1))
	} else {
		i.stack.Push(big.NewInt(0))
	}
}
