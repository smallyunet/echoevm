package core

import "math/big"

type Memory struct {
	data []byte
}

func NewMemory() *Memory {
	return &Memory{data: make([]byte, 0)}
}

func (m *Memory) Resize(size uint64) {
	if uint64(len(m.data)) >= size {
		return
	}
	// Expand to multiple of 32
	newSize := (size + 31) / 32 * 32
	needed := newSize - uint64(len(m.data))
	if needed > 0 {
		m.data = append(m.data, make([]byte, needed)...)
	}
}

func (m *Memory) Set(offset uint64, value *big.Int) {
	// Ensure memory size is at least offset + 32
	m.Resize(offset + 32)
	// Write 32-byte big-endian value
	value.FillBytes(m.data[offset : offset+32])
}

func (m *Memory) Get(offset uint64) []byte {
	m.Resize(offset + 32)
	return m.data[offset : offset+32]
}

func (m *Memory) Write(offset uint64, data []byte) {
	if len(data) == 0 {
		return
	}
	m.Resize(offset + uint64(len(data)))
	copy(m.data[offset:offset+uint64(len(data))], data)
}

// Read returns a copy of `size` bytes starting at `offset`.
// Bytes beyond the current memory length are zero-filled.
func (m *Memory) Read(offset, size uint64) []byte {
	end := offset + size
	out := make([]byte, size)
	if offset < uint64(len(m.data)) {
		copy(out, m.data[offset:min(end, uint64(len(m.data)))])
	}
	return out
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
func (m *Memory) Len() int {
	return len(m.data)
}

func (m *Memory) Data() []byte {
	return m.data
}

// Copy copies length bytes from src to dest.
// Both src and dest are offsets in memory.
// It handles memory expansion if necessary (caller should have checked gas).
func (m *Memory) Copy(dest, src, length uint64) {
	if length == 0 {
		return
	}
	// Resize for the furthest point
	maxEnd := dest + length
	if src+length > maxEnd {
		maxEnd = src + length
	}
	m.Resize(maxEnd)
	
	// Use copy built-in which handles overlap correctly
	copy(m.data[dest:dest+length], m.data[src:src+length])
}
