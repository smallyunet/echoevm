package rpc

import (
	"runtime"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// Web3API offers web3 related RPC methods
type Web3API struct {
	server *Server
}

// NewWeb3API creates a new instance of Web3API
func NewWeb3API(server *Server) *Web3API {
	return &Web3API{
		server: server,
	}
}

// ClientVersion returns the current client version
func (api *Web3API) ClientVersion() string {
	return "echoevm/" + runtime.Version()
}

// Sha3 returns the keccak-256 hash of the given data
func (api *Web3API) Sha3(input hexutil.Bytes) hexutil.Bytes {
	return hexutil.Bytes(crypto.Keccak256(input))
}

// NetAPI offers network related RPC methods
type NetAPI struct {
	server *Server
}

// NewNetAPI creates a new instance of NetAPI
func NewNetAPI(server *Server) *NetAPI {
	return &NetAPI{
		server: server,
	}
}

// Version returns the current network ID (chain ID)
func (api *NetAPI) Version() string {
	// Default to mainnet chain ID (1)
	return "1"
}

// Listening returns if client is actively listening for network connections
func (api *NetAPI) Listening() bool {
	return true // always listening
}

// PeerCount returns the number of connected peers
func (api *NetAPI) PeerCount() hexutil.Uint {
	return hexutil.Uint(0) // No peers in standalone mode
}
