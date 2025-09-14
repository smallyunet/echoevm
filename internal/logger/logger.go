package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	configpkg "github.com/smallyunet/echoevm/internal/config"
	"github.com/smallyunet/echoevm/internal/errors"
)

// Logger wraps zerolog.Logger with additional context and structured logging
type Logger struct {
	zerolog.Logger
	component string
	version   string
}

// Config holds logger configuration
type Config struct {
	Level      string
	Format     string // "json" or "console"
	Output     string // "stdout", "stderr", or file path
	Component  string
	Version    string
	TimeFormat string
}

// DefaultConfig returns a default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "console",
		Output:     "stdout",
		Component:  "echoevm",
		Version:    "1.0.0",
		TimeFormat: time.RFC3339,
	}
}

// New creates a new logger with the given configuration
func New(config *Config) (*Logger, error) {
	// Parse log level
	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	// Set global level
	zerolog.SetGlobalLevel(level)

	// Create base logger
	var baseLogger zerolog.Logger

	// Configure output
	var output zerolog.ConsoleWriter
	switch config.Output {
	case "stdout":
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: config.TimeFormat}
	case "stderr":
		output = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: config.TimeFormat}
	default:
		// When output is neither stdout nor stderr we treat it as a file path.
		// Use the default file permission defined in the config constants package.
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, configpkg.DefaultLogFileMode)
		if err != nil {
			return nil, errors.NewLogError(errors.LogLevelError, "failed to open log file", config.Component)
		}
		output = zerolog.ConsoleWriter{Out: file, TimeFormat: config.TimeFormat}
	}

	// Configure format
	if config.Format == "json" {
		baseLogger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		baseLogger = zerolog.New(output).With().Timestamp().Logger()
	}

	// Add component and version context
	logger := baseLogger.With().
		Str("component", config.Component).
		Str("version", config.Version).
		Logger()

	return &Logger{
		Logger:    logger,
		component: config.Component,
		version:   config.Version,
	}, nil
}

// WithContext creates a new logger with additional context
func (l *Logger) WithContext(fields map[string]interface{}) *Logger {
	event := l.Logger.With()
	for k, v := range fields {
		event = event.Interface(k, v)
	}

	return &Logger{
		Logger:    event.Logger(),
		component: l.component,
		version:   l.version,
	}
}

// WithComponent creates a new logger with a specific component
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger:    l.Logger.With().Str("component", component).Logger(),
		component: component,
		version:   l.version,
	}
}

// WithRequestID creates a new logger with a request ID
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger:    l.Logger.With().Str("request_id", requestID).Logger(),
		component: l.component,
		version:   l.version,
	}
}

// WithTransaction creates a new logger with transaction context
func (l *Logger) WithTransaction(txHash string, blockNumber uint64) *Logger {
	return &Logger{
		Logger:    l.Logger.With().Str("tx_hash", txHash).Uint64("block_number", blockNumber).Logger(),
		component: l.component,
		version:   l.version,
	}
}

// WithEVMContext creates a new logger with EVM execution context
func (l *Logger) WithEVMContext(pc uint64, opcode byte, gas uint64) *Logger {
	return &Logger{
		Logger:    l.Logger.With().Uint64("pc", pc).Uint8("opcode", opcode).Uint64("gas", gas).Logger(),
		component: l.component,
		version:   l.version,
	}
}

// EVMExecution logs EVM execution events
func (l *Logger) EVMExecution(pc uint64, opcode byte, stack []string, gas uint64) {
	l.Debug().
		Uint64("pc", pc).
		Uint8("opcode", opcode).
		Str("opcode_name", getOpcodeName(opcode)).
		Strs("stack", stack).
		Uint64("gas", gas).
		Msg("EVM execution step")
}

// EVMError logs EVM execution errors
func (l *Logger) EVMError(pc uint64, opcode byte, err error, stack []string) {
	l.Error().
		Uint64("pc", pc).
		Uint8("opcode", opcode).
		Str("opcode_name", getOpcodeName(opcode)).
		Err(err).
		Strs("stack", stack).
		Msg("EVM execution error")
}

// StackOperation logs stack operations
func (l *Logger) StackOperation(operation string, stackSize int, value string) {
	l.Trace().
		Str("operation", operation).
		Int("stack_size", stackSize).
		Str("value", value).
		Msg("Stack operation")
}

// StackError logs stack errors
func (l *Logger) StackError(operation string, err error, stackSize int) {
	l.Error().
		Str("operation", operation).
		Err(err).
		Int("stack_size", stackSize).
		Msg("Stack error")
}

// MemoryOperation logs memory operations
func (l *Logger) MemoryOperation(operation string, offset, size uint64, value string) {
	l.Trace().
		Str("operation", operation).
		Uint64("offset", offset).
		Uint64("size", size).
		Str("value", value).
		Msg("Memory operation")
}

// StorageOperation logs storage operations
func (l *Logger) StorageOperation(operation string, key, value string) {
	l.Trace().
		Str("operation", operation).
		Str("key", key).
		Str("value", value).
		Msg("Storage operation")
}

// RPCRequest logs incoming RPC requests
func (l *Logger) RPCRequest(method, requestID string, params interface{}) {
	l.Info().
		Str("method", method).
		Str("request_id", requestID).
		Interface("params", params).
		Msg("RPC request received")
}

// RPCResponse logs RPC responses
func (l *Logger) RPCResponse(method, requestID string, result interface{}, duration time.Duration) {
	l.Info().
		Str("method", method).
		Str("request_id", requestID).
		Interface("result", result).
		Dur("duration", duration).
		Msg("RPC response sent")
}

// RPCError logs RPC errors
func (l *Logger) RPCError(method, requestID string, err error, duration time.Duration) {
	l.Error().
		Str("method", method).
		Str("request_id", requestID).
		Err(err).
		Dur("duration", duration).
		Msg("RPC error")
}

// ContractExecution logs contract execution events
func (l *Logger) ContractExecution(contractAddress, function string, input []byte, gas uint64) {
	l.Info().
		Str("contract_address", contractAddress).
		Str("function", function).
		Bytes("input", input).
		Uint64("gas", gas).
		Msg("Contract execution started")
}

// ContractResult logs contract execution results
func (l *Logger) ContractResult(contractAddress string, output []byte, gasUsed uint64, success bool) {
	level := l.Logger.Info()
	if !success {
		level = l.Logger.Error()
	}

	level.
		Str("contract_address", contractAddress).
		Bytes("output", output).
		Uint64("gas_used", gasUsed).
		Bool("success", success).
		Msg("Contract execution completed")
}

// BlockProcessing logs block processing events
func (l *Logger) BlockProcessing(blockNumber uint64, txCount int) {
	l.Info().
		Uint64("block_number", blockNumber).
		Int("transaction_count", txCount).
		Msg("Block processing started")
}

// BlockCompleted logs block completion
func (l *Logger) BlockCompleted(blockNumber uint64, successCount, totalCount int, duration time.Duration) {
	l.Info().
		Uint64("block_number", blockNumber).
		Int("success_count", successCount).
		Int("total_count", totalCount).
		Dur("duration", duration).
		Msg("Block processing completed")
}

// Configuration logs configuration events
func (l *Logger) Configuration(field string, value interface{}) {
	l.Debug().
		Str("field", field).
		Interface("value", value).
		Msg("Configuration loaded")
}

// ConfigurationError logs configuration errors
func (l *Logger) ConfigurationError(field string, value interface{}, err error) {
	l.Error().
		Str("field", field).
		Interface("value", value).
		Err(err).
		Msg("Configuration error")
}

// Startup logs application startup
func (l *Logger) Startup(version, buildTime string) {
	l.Info().
		Str("version", version).
		Str("build_time", buildTime).
		Msg("Application starting")
}

// Shutdown logs application shutdown
func (l *Logger) Shutdown(reason string) {
	l.Info().
		Str("reason", reason).
		Msg("Application shutting down")
}

// Performance logs performance metrics
func (l *Logger) Performance(operation string, duration time.Duration, metadata map[string]interface{}) {
	event := l.Debug().
		Str("operation", operation).
		Dur("duration", duration)

	for k, v := range metadata {
		event = event.Interface(k, v)
	}

	event.Msg("Performance metric")
}

// Security logs security-related events
func (l *Logger) Security(event string, details map[string]interface{}) {
	eventLogger := l.Logger.Warn().
		Str("security_event", event)

	for k, v := range details {
		eventLogger = eventLogger.Interface(k, v)
	}

	eventLogger.Msg("Security event")
}

// getOpcodeName returns the name of an opcode
func getOpcodeName(opcode byte) string {
	// This is a simplified version - you might want to import the actual opcode names
	opcodeNames := map[byte]string{
		0x00: "STOP",
		0x01: "ADD",
		0x02: "MUL",
		0x03: "SUB",
		0x04: "DIV",
		0x05: "SDIV",
		0x06: "MOD",
		0x07: "SMOD",
		0x08: "ADDMOD",
		0x09: "MULMOD",
		0x0a: "EXP",
		0x0b: "SIGNEXTEND",
		0x10: "LT",
		0x11: "GT",
		0x12: "SLT",
		0x13: "SGT",
		0x14: "EQ",
		0x15: "ISZERO",
		0x16: "AND",
		0x17: "OR",
		0x18: "XOR",
		0x19: "NOT",
		0x1a: "BYTE",
		0x1b: "SHL",
		0x1c: "SHR",
		0x1d: "SAR",
		0x20: "SHA3",
		0x30: "ADDRESS",
		0x31: "BALANCE",
		0x32: "ORIGIN",
		0x33: "CALLER",
		0x34: "CALLVALUE",
		0x35: "CALLDATALOAD",
		0x36: "CALLDATASIZE",
		0x37: "CALLDATACOPY",
		0x38: "CODESIZE",
		0x39: "CODECOPY",
		0x3a: "GASPRICE",
		0x3b: "EXTCODESIZE",
		0x3c: "EXTCODECOPY",
		0x3d: "RETURNDATASIZE",
		0x3e: "RETURNDATACOPY",
		0x3f: "EXTCODEHASH",
		0x40: "BLOCKHASH",
		0x41: "COINBASE",
		0x42: "TIMESTAMP",
		0x43: "NUMBER",
		0x44: "DIFFICULTY",
		0x45: "GASLIMIT",
		0x46: "CHAINID",
		0x47: "SELFBALANCE",
		0x48: "BASEFEE",
		0x50: "POP",
		0x51: "MLOAD",
		0x52: "MSTORE",
		0x53: "MSTORE8",
		0x54: "SLOAD",
		0x55: "SSTORE",
		0x56: "JUMP",
		0x57: "JUMPI",
		0x58: "PC",
		0x59: "MSIZE",
		0x5a: "GAS",
		0x5b: "JUMPDEST",
		0x5f: "PUSH0",
		0x60: "PUSH1",
		0x61: "PUSH2",
		0x62: "PUSH3",
		0x63: "PUSH4",
		0x64: "PUSH5",
		0x65: "PUSH6",
		0x66: "PUSH7",
		0x67: "PUSH8",
		0x68: "PUSH9",
		0x69: "PUSH10",
		0x6a: "PUSH11",
		0x6b: "PUSH12",
		0x6c: "PUSH13",
		0x6d: "PUSH14",
		0x6e: "PUSH15",
		0x6f: "PUSH16",
		0x70: "PUSH17",
		0x71: "PUSH18",
		0x72: "PUSH19",
		0x73: "PUSH20",
		0x74: "PUSH21",
		0x75: "PUSH22",
		0x76: "PUSH23",
		0x77: "PUSH24",
		0x78: "PUSH25",
		0x79: "PUSH26",
		0x7a: "PUSH27",
		0x7b: "PUSH28",
		0x7c: "PUSH29",
		0x7d: "PUSH30",
		0x7e: "PUSH31",
		0x7f: "PUSH32",
		0x80: "DUP1",
		0x81: "DUP2",
		0x82: "DUP3",
		0x83: "DUP4",
		0x84: "DUP5",
		0x85: "DUP6",
		0x86: "DUP7",
		0x87: "DUP8",
		0x88: "DUP9",
		0x89: "DUP10",
		0x8a: "DUP11",
		0x8b: "DUP12",
		0x8c: "DUP13",
		0x8d: "DUP14",
		0x8e: "DUP15",
		0x8f: "DUP16",
		0x90: "SWAP1",
		0x91: "SWAP2",
		0x92: "SWAP3",
		0x93: "SWAP4",
		0x94: "SWAP5",
		0x95: "SWAP6",
		0x96: "SWAP7",
		0x97: "SWAP8",
		0x98: "SWAP9",
		0x99: "SWAP10",
		0x9a: "SWAP11",
		0x9b: "SWAP12",
		0x9c: "SWAP13",
		0x9d: "SWAP14",
		0x9e: "SWAP15",
		0x9f: "SWAP16",
		0xa0: "LOG0",
		0xa1: "LOG1",
		0xa2: "LOG2",
		0xa3: "LOG3",
		0xa4: "LOG4",
		0xf0: "CREATE",
		0xf1: "CALL",
		0xf2: "CALLCODE",
		0xf3: "RETURN",
		0xf4: "DELEGATECALL",
		0xf5: "CREATE2",
		0xfa: "STATICCALL",
		0xfd: "REVERT",
		0xfe: "INVALID",
		0xff: "SELFDESTRUCT",
	}

	if name, ok := opcodeNames[opcode]; ok {
		return name
	}
	return "UNKNOWN"
}

// Global logger instance
var globalLogger *Logger

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		// Create a default logger if none is set
		config := DefaultConfig()
		logger, err := New(config)
		if err != nil {
			// Fallback to zerolog's global logger
			return &Logger{Logger: log.Logger}
		}
		globalLogger = logger
	}
	return globalLogger
}
