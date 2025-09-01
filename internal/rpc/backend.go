package rpc

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

// EchoEvmBackend provides the backend functionality for RPC methods
type EchoEvmBackend struct {
	blockNumber uint64
	codeStorage map[common.Address][]byte
}

// NewEchoEvmBackend creates a new backend instance
func NewEchoEvmBackend() *EchoEvmBackend {
	return &EchoEvmBackend{
		blockNumber: 1,
		codeStorage: make(map[common.Address][]byte),
	}
}

// GetLatestBlockNumber returns the current block number
func (b *EchoEvmBackend) GetLatestBlockNumber() uint64 {
	return b.blockNumber
}

// GetCodeAt returns the code at the given address
func (b *EchoEvmBackend) GetCodeAt(address *common.Address) []byte {
	if address == nil {
		return nil
	}
	return b.codeStorage[*address]
}

// SetCode sets the code for a specific address
func (b *EchoEvmBackend) SetCode(address common.Address, code []byte) {
	b.codeStorage[address] = code
}

// BlockNumberOrHash represents a block number or a block hash
type BlockNumberOrHash = rpc.BlockNumberOrHash

// BlockNumber represents a block number or a special block reference
type BlockNumber = rpc.BlockNumber

const (
	// LatestBlockNumber is used when the latest known block should be used
	LatestBlockNumber = rpc.LatestBlockNumber

	// PendingBlockNumber is used when the pending block should be used
	PendingBlockNumber = rpc.PendingBlockNumber

	// EarliestBlockNumber is used when the earliest known block should be used
	EarliestBlockNumber = rpc.EarliestBlockNumber
)
