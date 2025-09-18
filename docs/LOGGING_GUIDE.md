# Logging Configuration for EchoEVM

## Overview
This document describes the logging system improvements made to EchoEVM to enhance reliability and debugging capabilities.

## Log Levels

### TRACE (0)
- **Usage**: Detailed execution tracing
- **Content**: 
  - EVM opcode execution steps
  - Stack operations (push/pop/dup/swap)
  - Memory operations (load/store)
  - Storage operations (sload/sstore)
- **When to use**: Debugging complex contract execution issues

### DEBUG (1)
- **Usage**: Detailed debugging information
- **Content**:
  - Bytecode disassembly
  - Configuration loading
  - Performance metrics
- **When to use**: Development and testing

### INFO (2)
- **Usage**: General operational information
- **Content**:
  - Application startup/shutdown
  - Contract execution results
  - Block processing status
- **When to use**: Normal operation monitoring

### WARN (3)
- **Usage**: Warning conditions
- **Content**:
  - Missing contract code
  - Deprecated features
  - Performance warnings
- **When to use**: Issues that don't stop execution but need attention

### ERROR (4)
- **Usage**: Error conditions
- **Content**:
  - EVM execution errors
  - Stack errors (overflow/underflow)
  - Configuration errors
- **When to use**: Issues that affect functionality

### FATAL (5)
- **Usage**: Critical errors
- **Content**:
  - Startup failures
  - Critical configuration errors
  - Unrecoverable system errors
- **When to use**: Issues that require immediate attention and stop execution

## Structured Logging Fields

### Common Fields
- `component`: The component generating the log (e.g., "echoevm", "rpc", "vm")
- `version`: Application version
- `timestamp`: Log timestamp in RFC3339 format

### EVM Execution Fields
- `pc`: Program counter (uint64)
- `pc_hex`: Program counter in hex format (string)
- `opcode`: Opcode byte value (uint8)
- `opcode_name`: Human-readable opcode name (string)
- `stack_size`: Current stack size (int)
- `stack`: Stack contents as hex strings ([]string)
- `gas`: Gas remaining (uint64)

### Transaction Fields
- `tx_hash`: Transaction hash (string)
- `block_number`: Block number (uint64)
- `contract_address`: Contract address (string)
- `function`: Function name (string)
- `gas_used`: Gas consumed (uint64)

### Error Fields
- `error`: Error message (string)
- `error_type`: Type of error (string)
- `stack_trace`: Stack trace for errors (string)

## Log Output Formats

### Console Format (Default)
```
2024-01-15T10:30:45.123Z INF EVM execution step component=echoevm pc=0x0000 opcode=0x60 opcode_name=PUSH1 stack_size=0
2024-01-15T10:30:45.124Z INF RETURN operation executed component=echoevm offset=0x00 size=0x20 return_data_hex=0x1234... return_data_size=32
```

### JSON Format
```json
{
  "level": "info",
  "component": "echoevm",
  "version": "1.0.0",
  "timestamp": "2024-01-15T10:30:45.123Z",
  "pc": 0,
  "pc_hex": "0x0000",
  "opcode": 96,
  "opcode_name": "PUSH1",
  "stack_size": 0,
  "message": "EVM execution step"
}
```

## Configuration

### Environment Variables
- `ECHOEVM_LOG_LEVEL`: Set log level (trace, debug, info, warn, error, fatal)
- `ECHOEVM_LOG_FORMAT`: Set output format (console, json)
- `ECHOEVM_LOG_OUTPUT`: Set output destination (stdout, stderr, file path)

### Command Line Flags
- `--log-level`: Set log level
- `--log-format`: Set output format
- `--log-file`: Set log file path

## Best Practices

### 1. Use Appropriate Log Levels
- Use TRACE for detailed execution tracing
- Use DEBUG for development debugging
- Use INFO for normal operation monitoring
- Use WARN for potential issues
- Use ERROR for actual errors
- Use FATAL only for critical failures

### 2. Include Relevant Context
- Always include component and version
- Include relevant IDs (request_id, tx_hash, etc.)
- Include performance metrics when relevant
- Include error details with proper error handling

### 3. Structured Logging
- Use structured fields instead of string interpolation
- Use consistent field names across components
- Include both human-readable and machine-readable formats

### 4. Performance Considerations
- Use TRACE level sparingly in production
- Consider log rotation for file output
- Use appropriate sampling for high-volume operations

### 5. Security
- Never log sensitive data (private keys, passwords)
- Be careful with user input in logs
- Consider log sanitization for external systems

## Migration Guide

### From Old Logging System
1. Replace `fmt.Printf` with structured logging
2. Replace `log.Println` with appropriate log levels
3. Add context fields to existing log messages
4. Update error handling to include structured error information

### Example Migration
```go
// Old way
fmt.Printf("Executing opcode 0x%02x at PC 0x%04x\n", opcode, pc)

// New way
logger.Trace().
    Uint8("opcode", opcode).
    Str("opcode_hex", fmt.Sprintf("0x%02x", opcode)).
    Uint64("pc", pc).
    Str("pc_hex", fmt.Sprintf("0x%04x", pc)).
    Msg("Executing opcode")
```

## Monitoring and Alerting

### Key Metrics to Monitor
- Error rate by component
- EVM execution success rate
- Memory and stack usage patterns

### Alerting Rules
- Error rate > 5% over 5 minutes
- EVM execution failures > 10% over 1 minute
- Memory usage > 80% of available

## Troubleshooting

### Common Issues
1. **High log volume**: Increase log level or implement sampling
2. **Missing context**: Ensure all log calls include relevant fields
3. **Performance impact**: Use appropriate log levels and consider async logging
4. **Storage issues**: Implement log rotation and compression

### Debug Commands
```bash
# Set debug level for detailed logging
export ECHOEVM_LOG_LEVEL=debug

# Use JSON format for log aggregation
export ECHOEVM_LOG_FORMAT=json

# Redirect logs to file
export ECHOEVM_LOG_OUTPUT=/var/log/echoevm.log
```
