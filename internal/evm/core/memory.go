package core

import "math/big"

type Memory struct {
	data []byte
}

func NewMemory() *Memory {
	return &Memory{data: make([]byte, 0)}
}

func (m *Memory) Set(offset uint64, value *big.Int) {
	// Ensure memory size is at least offset + 32
	end := offset + 32
	if uint64(len(m.data)) < end {
		newMem := make([]byte, end)
		copy(newMem, m.data)
		m.data = newMem
	}
	// Write 32-byte big-endian value
	bytes := value.FillBytes(make([]byte, 32))
	copy(m.data[offset:end], bytes)
}

func (m *Memory) Get(offset uint64) []byte {
	end := offset + 32
	if uint64(len(m.data)) < end {
		return make([]byte, 32)
	}
	return m.data[offset:end]
}

func (m *Memory) Write(offset uint64, data []byte) {
	end := offset + uint64(len(data))
	if uint64(len(m.data)) < end {
		newData := make([]byte, end)
		copy(newData, m.data)
		m.data = newData
	}
	copy(m.data[offset:end], data)
}
