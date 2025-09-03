package config

import "time"

// EVM Constants
const (
	// StackLimit is the maximum number of items that can be stored on the EVM stack
	StackLimit = 1024

	// DefaultGasLimit is the default gas limit for transactions
	DefaultGasLimit = 15000000

	// DefaultBlockGasLimit is the default gas limit for blocks
	DefaultBlockGasLimit = 15000000

	// LogsBloomSize is the size of the logs bloom filter
	LogsBloomSize = 256

	// DefaultTimestamp is the default timestamp for blocks (2022-01-01 00:00:00 UTC)
	DefaultTimestamp = 1640995200
)

// RPC Constants
const (
	// DefaultRPCEndpoint is the default HTTP RPC endpoint
	DefaultRPCEndpoint = "localhost:8545"

	// DefaultRPCTimeout is the default timeout for RPC calls
	DefaultRPCTimeout = 30 * time.Second

	// DefaultRPCReadTimeout is the default read timeout for RPC server
	DefaultRPCReadTimeout = 15 * time.Second

	// DefaultRPCWriteTimeout is the default write timeout for RPC server
	DefaultRPCWriteTimeout = 15 * time.Second

	// DefaultRPCIdleTimeout is the default idle timeout for RPC server
	DefaultRPCIdleTimeout = 60 * time.Second
)

// Network Constants
const (
	// DefaultEthereumRPC is the default Ethereum RPC endpoint
	DefaultEthereumRPC = "https://cloudflare-eth.com"

	// DefaultChainID is the default chain ID
	DefaultChainID = 1
)

// Logging Constants
const (
	// DefaultLogLevel is the default log level
	DefaultLogLevel = "info"

	// DefaultLogFormat is the default log format
	DefaultLogFormat = "console"

	// DefaultLogOutput is the default log output
	DefaultLogOutput = "stdout"

	// DefaultLogComponent is the default log component name
	DefaultLogComponent = "echoevm"

	// DefaultLogVersion is the default log version
	DefaultLogVersion = "1.0.0"

	// DefaultLogTimeFormat is the default log time format
	DefaultLogTimeFormat = time.RFC3339

	// DefaultLogFileMode is the default file mode for log files
	DefaultLogFileMode = 0666
)

// API Constants
const (
	// DefaultAPINamespace is the default API namespace
	DefaultAPINamespace = "eth"

	// DefaultAPIVersion is the default API version
	DefaultAPIVersion = "1.0"

	// DefaultAPIPublic is the default API public flag
	DefaultAPIPublic = true
)

// Execution Constants
const (
	// DefaultExecutionMode is the default execution mode
	DefaultExecutionMode = "full"

	// DefaultGasPrice is the default gas price
	DefaultGasPrice = 20000000000 // 20 gwei

	// DefaultValue is the default transaction value
	DefaultValue = 0
)

// File Constants
const (
	// DefaultFileMode is the default file mode
	DefaultFileMode = 0644

	// DefaultDirectoryMode is the default directory mode
	DefaultDirectoryMode = 0755
)
