package core

import "math/big"

type Stack struct{ data []*big.Int }

func NewStack() *Stack           { return &Stack{} }
func (s *Stack) Push(x *big.Int) { s.data = append(s.data, x) }
func (s *Stack) Pop() *big.Int   { v := s.data[len(s.data)-1]; s.data = s.data[:len(s.data)-1]; return v }
func (s *Stack) Len() int        { return len(s.data) }
