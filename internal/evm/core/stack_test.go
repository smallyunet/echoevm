package core

import (
	"math/big"
	"testing"
)

func TestStackPushPop(t *testing.T) {
	s := NewStack()
	s.Push(big.NewInt(1))
	if s.Len() != 1 {
		t.Fatalf("expected length 1, got %d", s.Len())
	}
	v := s.Pop()
	if v.Int64() != 1 {
		t.Fatalf("expected 1, got %s", v)
	}
	if s.Len() != 0 {
		t.Fatalf("expected empty stack")
	}
}

func TestStackPeek(t *testing.T) {
	s := NewStack()
	s.Push(big.NewInt(1))
	s.Push(big.NewInt(2))
	if s.Peek(0).Int64() != 2 {
		t.Fatalf("peek top failed")
	}
	if s.Peek(1).Int64() != 1 {
		t.Fatalf("peek second failed")
	}
}

func TestStackDupSwap(t *testing.T) {
	s := NewStack()
	s.Push(big.NewInt(1))
	s.Push(big.NewInt(2))
	s.Dup(1)
	if s.Peek(0).Int64() != 2 {
		t.Fatalf("dup failed")
	}
	s.Swap(2)
	if s.Peek(0).Int64() != 1 {
		t.Fatalf("swap failed")
	}
}

func TestStackUnderflow(t *testing.T) {
	defer func() { recover() }()
	s := NewStack()
	s.Pop()
	t.Fatal("expected panic on underflow")
}

func TestStackOverflow(t *testing.T) {
	defer func() { recover() }()
	s := NewStack()
	for i := 0; i < StackLimit+1; i++ {
		s.Push(big.NewInt(int64(i)))
	}
	t.Fatal("expected panic on overflow")
}
