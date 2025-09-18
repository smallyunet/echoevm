# EchoEVM Configuration Guide

This document describes the configuration options available in EchoEVM and how to customize them.

## Configuration Sources

EchoEVM supports configuration through multiple sources in the following order of precedence:

1. Command line flags (highest priority)
2. Environment variables
3. Default constants (lowest priority)

## Environment Variables

You can override default configuration values using the following environment variables:

### Network Configuration
- `ECHOEVM_ETHEREUM_RPC`: Ethereum RPC endpoint for block/range commands (default: `https://cloudflare-eth.com`)

### Logging Configuration
- `ECHOEVM_LOG_LEVEL`: Log level (default: `info`)
- `ECHOEVM_LOG_FORMAT`: Log format - "json" or "console" (default: `console`)
- `ECHOEVM_LOG_OUTPUT`: Log output - "stdout", "stderr", or file path (default: `stdout`)

### EVM Configuration
- `ECHOEVM_GAS_LIMIT`: Default gas limit for transactions (default: `15000000`)
- `ECHOEVM_BLOCK_GAS_LIMIT`: Default gas limit for blocks (default: `15000000`)
- `ECHOEVM_CHAIN_ID`: Default chain ID (default: `1`)
- `ECHOEVM_EXECUTION_MODE`: Default execution mode (default: `full`)

### API Configuration
- `ECHOEVM_API_NAMESPACE`: Default API namespace (default: `eth`)
- `ECHOEVM_API_VERSION`: Default API version (default: `1.0`)
- `ECHOEVM_API_PUBLIC`: Default API public flag (default: `true`)

## Usage Examples

### Setting Log Level
```bash
export ECHOEVM_LOG_LEVEL="debug"
./echoevm run -bin contract.bin
```

### Setting Gas Limit
```bash
export ECHOEVM_GAS_LIMIT="30000000"
./echoevm run -bin contract.bin
```

### Setting Ethereum RPC Endpoint
```bash
export ECHOEVM_ETHEREUM_RPC="https://mainnet.infura.io/v3/<key>"
./echoevm block --block 19000000
```

### Using JSON Logging
```bash
export ECHOEVM_LOG_FORMAT="json"
export ECHOEVM_LOG_LEVEL="debug"
./echoevm run -bin contract.bin
```

## Configuration Constants

The following constants are defined in `internal/config/constants.go`:

### EVM Constants
- `StackLimit`: Maximum stack size (1024)
- `DefaultGasLimit`: Default transaction gas limit (15000000)
- `DefaultBlockGasLimit`: Default block gas limit (15000000)
- `LogsBloomSize`: Logs bloom filter size (256)
- `DefaultTimestamp`: Default block timestamp (1640995200)

### Network Constants
- `DefaultEthereumRPC`: Default Ethereum RPC endpoint ("https://cloudflare-eth.com")
- `DefaultChainID`: Default chain ID (1)

### Logging Constants
- `DefaultLogLevel`: Default log level ("info")
- `DefaultLogFormat`: Default log format ("console")
- `DefaultLogOutput`: Default log output ("stdout")
- `DefaultLogComponent`: Default log component ("echoevm")
- `DefaultLogVersion`: Default log version ("1.0.0")
- `DefaultLogTimeFormat`: Default log time format (time.RFC3339)
- `DefaultLogFileMode`: Default log file mode (0666)

### API Constants
- `DefaultAPINamespace`: Default API namespace ("eth")
- `DefaultAPIVersion`: Default API version ("1.0")
- `DefaultAPIPublic`: Default API public flag (true)

### Execution Constants
- `DefaultExecutionMode`: Default execution mode ("full")
- `DefaultGasPrice`: Default gas price (20000000000)
- `DefaultValue`: Default transaction value (0)

### File Constants
- `DefaultFileMode`: Default file mode (0644)
- `DefaultDirectoryMode`: Default directory mode (0755)

## Best Practices

1. **Use Environment Variables**: For production deployments, use environment variables to configure the application rather than hardcoding values.

2. **Logging**: Use JSON format for production logging to enable better log aggregation and analysis.

3. **Security**: Be careful with file permissions when logging to files. The default file mode is 0666, but you may want to use more restrictive permissions in production.

4. **Performance**: Adjust gas limits and upstream RPC endpoints based on your specific use case and network conditions.

5. **Monitoring**: Use appropriate log levels to balance between debugging information and performance.

## Migration from Hardcoded Values

If you're migrating from a version with hardcoded values, you can:

1. Set environment variables to match your previous hardcoded values
2. Gradually adjust them based on your new requirements
3. Use the configuration system to make your deployment more flexible

## Troubleshooting

### Configuration Not Applied
- Check that environment variables are set before running the command
- Verify that the environment variable names are correct (case-sensitive)
- Ensure that the values are in the correct format

### Invalid Values
- Gas limits must be positive integers
- Boolean values must be "true" or "false"
- Log levels must be one of: "trace", "debug", "info", "warn", "error"

### Performance Issues
- Increase gas limits if transactions are failing due to out-of-gas errors
- Use appropriate log levels to reduce logging overhead
