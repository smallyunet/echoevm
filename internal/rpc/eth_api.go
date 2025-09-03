package rpc

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/smallyunet/echoevm/internal/config"
	"github.com/smallyunet/echoevm/internal/evm/vm"
)

// EthAPI offers ethereum related RPC methods
type EthAPI struct {
	server  *Server
	backend *EchoEvmBackend
}

// NewEthAPI creates a new instance of EthAPI
func NewEthAPI(server *Server) *EthAPI {
	return &EthAPI{
		server:  server,
		backend: NewEchoEvmBackend(),
	}
}

// BlockNumber returns the latest block number
func (api *EthAPI) BlockNumber() (hexutil.Uint64, error) {
	blockNumber := api.backend.GetLatestBlockNumber()
	return hexutil.Uint64(blockNumber), nil
}

// GetBalance returns the account balance for the given account
func (api *EthAPI) GetBalance(address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	// In a minimal implementation, we can return a zero balance for all accounts
	return (*hexutil.Big)(big.NewInt(0)), nil
}

// GetBlockByNumber returns the requested block
func (api *EthAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	// Fetch block from RPC or use internal state
	// For now, we'll provide a minimal implementation

	block := make(map[string]interface{})
	blockNumber := uint64(number.Int64())
	if number == rpc.LatestBlockNumber {
		blockNumber = api.backend.GetLatestBlockNumber()
	}

	block["number"] = hexutil.Uint64(blockNumber)
	block["hash"] = common.Hash{} // Empty hash
	block["parentHash"] = common.Hash{}
	block["nonce"] = hexutil.Uint64(0)
	block["sha3Uncles"] = common.Hash{}
	block["logsBloom"] = hexutil.Bytes(make([]byte, config.LogsBloomSize))
	block["transactionsRoot"] = common.Hash{}
	block["stateRoot"] = common.Hash{}
	block["receiptsRoot"] = common.Hash{}
	block["miner"] = common.Address{}
	block["difficulty"] = (*hexutil.Big)(big.NewInt(0))
	block["totalDifficulty"] = (*hexutil.Big)(big.NewInt(0))
	block["extraData"] = hexutil.Bytes([]byte{})
	block["size"] = hexutil.Uint64(0)
	block["gasLimit"] = hexutil.Uint64(config.DefaultBlockGasLimit)
	block["gasUsed"] = hexutil.Uint64(0)
	block["timestamp"] = hexutil.Uint64(config.DefaultTimestamp)
	block["transactions"] = []interface{}{}
	block["uncles"] = []common.Hash{}

	return block, nil
}

// Call executes a new message call immediately without creating a transaction
func (api *EthAPI) Call(ctx context.Context, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	// Convert arguments to code and calldata
	code := api.backend.GetCodeAt(args.To)
	if len(code) == 0 {
		return nil, nil
	}

	var callData []byte
	if args.Data != nil {
		callData = *args.Data
	}

	// Execute the call using our EVM
	interpreter := vm.NewWithCallData(code, callData)
	interpreter.Run()

	// Return the results
	if interpreter.IsReverted() {
		return nil, nil
	}

	// Return the result as bytes
	result := interpreter.ReturnedCode()
	return hexutil.Bytes(result), nil
}

// SendTransaction sends a transaction
func (api *EthAPI) SendTransaction(ctx context.Context, args TransactionArgs) (common.Hash, error) {
	// Generate a random transaction hash - in a real implementation, this would create and process a tx
	hash := common.BytesToHash(crypto.Keccak256([]byte("tx")))
	return hash, nil
}

// GetCode returns the code at the given address
func (api *EthAPI) GetCode(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	code := api.backend.GetCodeAt(&address)
	return hexutil.Bytes(code), nil
}

// GetTransactionCount returns the number of transactions sent from the given address
func (api *EthAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Uint64, error) {
	count := hexutil.Uint64(0)
	return &count, nil
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash
func (api *EthAPI) GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	// Return a minimal receipt
	receipt := make(map[string]interface{})
	receipt["transactionHash"] = hash
	receipt["transactionIndex"] = hexutil.Uint64(0)
	receipt["blockHash"] = common.Hash{}
	receipt["blockNumber"] = hexutil.Uint64(0)
	receipt["from"] = common.Address{}
	receipt["to"] = nil
	receipt["cumulativeGasUsed"] = hexutil.Uint64(0)
	receipt["gasUsed"] = hexutil.Uint64(0)
	receipt["contractAddress"] = nil
	receipt["logs"] = []interface{}{}
	receipt["logsBloom"] = hexutil.Bytes(make([]byte, config.LogsBloomSize))
	receipt["status"] = hexutil.Uint64(1)

	return receipt, nil
}

// TransactionArgs represents the arguments to construct a new transaction
// or a message call.
type TransactionArgs struct {
	From     *common.Address `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	Data     *hexutil.Bytes  `json:"data"`
}
