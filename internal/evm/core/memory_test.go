package core

import (
	"math/big"
	"testing"
)

func TestMemorySetGet(t *testing.T) {
	m := NewMemory()
	val := big.NewInt(42)
	m.Set(0, val)
	got := new(big.Int).SetBytes(m.Get(0))
	if got.Cmp(val) != 0 {
		t.Fatalf("want %v got %v", val, got)
	}
}

func TestMemoryWrite(t *testing.T) {
	m := NewMemory()
	data := []byte{1, 2, 3}
	m.Write(0, data)
	got := m.Get(0)[:3]
	for i, b := range data {
		if got[i] != b {
			t.Fatalf("byte %d mismatch", i)
		}
	}
}
