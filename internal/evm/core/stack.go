package core

import (
	"fmt"
	"math/big"

	"github.com/smallyunet/echoevm/internal/errors"
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
func (s *Stack) Push(x *big.Int) error {
	if len(s.data) >= StackLimit {
		return errors.ErrStackOverflow
	}
	s.data = append(s.data, new(big.Int).Set(x)) // make a copy to prevent mutation
	return nil
}

// Pop removes and returns the top item of the stack.
func (s *Stack) Pop() (*big.Int, error) {
	if len(s.data) == 0 {
		return nil, errors.ErrStackUnderflow
	}
	v := s.data[len(s.data)-1]
	s.data = s.data[:len(s.data)-1]
	return v, nil
}

// Peek returns the N-th element from the top without removing it (0-based).
func (s *Stack) Peek(n int) (*big.Int, error) {
	if n >= len(s.data) {
		return nil, errors.NewStackPeekError(n, len(s.data))
	}
	return s.data[len(s.data)-1-n], nil
}

// Dup duplicates the N-th stack item (1-based, as in EVM spec).
func (s *Stack) Dup(n int) error {
	if n <= 0 || n > len(s.data) {
		return errors.NewStackDupError(n)
	}
	return s.Push(new(big.Int).Set(s.data[len(s.data)-n]))
}

// Swap swaps the top item with the N-th item below it (1-based, as in EVM spec).
func (s *Stack) Swap(n int) error {
	if n <= 0 || n >= len(s.data) {
		return errors.NewStackSwapError(n)
	}
	top := len(s.data) - 1
	s.data[top], s.data[top-n] = s.data[top-n], s.data[top]
	return nil
}

// Len returns the current number of items in the stack.
func (s *Stack) Len() int {
	return len(s.data)
}

// Snapshot returns a slice of hex strings representing the stack contents from bottom to top.
func (s *Stack) Snapshot() []string {
	snap := make([]string, len(s.data))
	for i, v := range s.data {
		snap[i] = fmt.Sprintf("0x%x", v)
	}
	return snap
}

// PushSafe is a backward-compatible wrapper that panics on error
func (s *Stack) PushSafe(x *big.Int) {
	if err := s.Push(x); err != nil {
		panic(err)
	}
}

// PopSafe is a backward-compatible wrapper that panics on error
func (s *Stack) PopSafe() *big.Int {
	val, err := s.Pop()
	if err != nil {
		panic(err)
	}
	return val
}

// PeekSafe is a backward-compatible wrapper that panics on error
func (s *Stack) PeekSafe(n int) *big.Int {
	val, err := s.Peek(n)
	if err != nil {
		panic(err)
	}
	return val
}

// DupSafe is a backward-compatible wrapper that panics on error
func (s *Stack) DupSafe(n int) {
	if err := s.Dup(n); err != nil {
		panic(err)
	}
}

// SwapSafe is a backward-compatible wrapper that panics on error
func (s *Stack) SwapSafe(n int) {
	if err := s.Swap(n); err != nil {
		panic(err)
	}
}
