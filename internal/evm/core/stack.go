package core

import (
	"fmt"
	"math/big"
)

const StackLimit = 1024 // EVM stack size limit

type Stack struct {
	data []*big.Int
}

// NewStack creates a new EVM stack.
func NewStack() *Stack {
	return &Stack{}
}

// Push pushes an item onto the stack.
func (s *Stack) Push(x *big.Int) {
	if len(s.data) >= StackLimit {
		panic("stack overflow")
	}
	s.data = append(s.data, new(big.Int).Set(x)) // make a copy to prevent mutation
}

// Pop removes and returns the top item of the stack.
func (s *Stack) Pop() *big.Int {
	if len(s.data) == 0 {
		panic("stack underflow")
	}
	v := s.data[len(s.data)-1]
	s.data = s.data[:len(s.data)-1]
	return v
}

// Peek returns the N-th element from the top without removing it (0-based).
func (s *Stack) Peek(n int) *big.Int {
	if n >= len(s.data) {
		panic(fmt.Sprintf("peek out of bounds: %d >= %d", n, len(s.data)))
	}
	return s.data[len(s.data)-1-n]
}

// Dup duplicates the N-th stack item (1-based, as in EVM spec).
func (s *Stack) Dup(n int) {
	if n <= 0 || n > len(s.data) {
		panic(fmt.Sprintf("dup out of bounds: n=%d", n))
	}
	s.Push(new(big.Int).Set(s.data[len(s.data)-n]))
}

// Swap swaps the top item with the N-th item below it (1-based, as in EVM spec).
func (s *Stack) Swap(n int) {
	if n <= 0 || n >= len(s.data) {
		panic(fmt.Sprintf("swap out of bounds: n=%d", n))
	}
	top := len(s.data) - 1
	s.data[top], s.data[top-n] = s.data[top-n], s.data[top]
}

// Len returns the current number of items in the stack.
func (s *Stack) Len() int {
	return len(s.data)
}
