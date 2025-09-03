package core

import (
	"math/big"
	"testing"
)

func TestStackPushPop(t *testing.T) {
	s := NewStack()
	err := s.Push(big.NewInt(1))
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if s.Len() != 1 {
		t.Fatalf("expected length 1, got %d", s.Len())
	}
	v, err := s.Pop()
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}
	if v.Int64() != 1 {
		t.Fatalf("expected 1, got %s", v)
	}
	if s.Len() != 0 {
		t.Fatalf("expected empty stack")
	}
}

func TestStackPeek(t *testing.T) {
	s := NewStack()
	err := s.Push(big.NewInt(1))
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	err = s.Push(big.NewInt(2))
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	v, err := s.Peek(0)
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}
	if v.Int64() != 2 {
		t.Fatalf("peek top failed")
	}
	v, err = s.Peek(1)
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}
	if v.Int64() != 1 {
		t.Fatalf("peek second failed")
	}
}

func TestStackDupSwap(t *testing.T) {
	s := NewStack()
	err := s.Push(big.NewInt(1))
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	err = s.Push(big.NewInt(2))
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	err = s.Dup(1)
	if err != nil {
		t.Fatalf("Dup failed: %v", err)
	}
	v, err := s.Peek(0)
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}
	if v.Int64() != 2 {
		t.Fatalf("dup failed")
	}
	err = s.Swap(2)
	if err != nil {
		t.Fatalf("Swap failed: %v", err)
	}
	v, err = s.Peek(0)
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}
	if v.Int64() != 1 {
		t.Fatalf("swap failed")
	}
}

func TestStackUnderflow(t *testing.T) {
	s := NewStack()
	_, err := s.Pop()
	if err == nil {
		t.Fatal("expected error on underflow")
	}
}

func TestStackOverflow(t *testing.T) {
	s := NewStack()
	for i := 0; i < StackLimit; i++ {
		err := s.Push(big.NewInt(int64(i)))
		if err != nil {
			t.Fatalf("Push failed at %d: %v", i, err)
		}
	}
	// This should fail
	err := s.Push(big.NewInt(999))
	if err == nil {
		t.Fatal("expected error on overflow")
	}
}
