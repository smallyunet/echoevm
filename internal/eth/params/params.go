package params

const (
	TxGas                 uint64 = 21000
	CallNewAccountGas     uint64 = 25000
	TxAuthTupleGas        uint64 = 12500
	TxTokenPerNonZeroByte uint64 = 4
	TxCostFloorPerToken   uint64 = 10
	CreateDataGas         uint64 = 200
	MaxCodeSize                  = 24_576
	MaxTxGas              uint64 = 16_777_216
	BlobTxMaxBlobs               = 6
	BlobTxMinBlobGasprice uint64 = 1
)

// MainnetChainConfig remains a marker accepted by the compatibility signer
// constructor. Fork selection is owned by internal/evm/core.
var MainnetChainConfig = struct{}{}
