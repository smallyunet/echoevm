package errors

import (
	"fmt"
	"runtime"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	LogLevelTrace LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelTrace:
		return "TRACE"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// EVM execution errors
type EVMError struct {
	Code    int
	Message string
	PC      uint64
	Opcode  byte
	Stack   []string
}

func (e *EVMError) Error() string {
	return fmt.Sprintf("EVM error %d at PC 0x%04x (opcode 0x%02x): %s", e.Code, e.PC, e.Opcode, e.Message)
}

// Stack errors
type StackError struct {
	Operation string
	Message   string
	StackSize int
	Requested int
}

func (e *StackError) Error() string {
	return fmt.Sprintf("stack %s error: %s (stack size: %d, requested: %d)", e.Operation, e.Message, e.StackSize, e.Requested)
}

// Configuration errors
type ConfigError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error for %s: %s (value: %v)", e.Field, e.Message, e.Value)
}

// RPC errors
type RPCError struct {
	Method    string
	Message   string
	RequestID string
	Code      int
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error %d in %s: %s (request: %s)", e.Code, e.Method, e.Message, e.RequestID)
}

// Logging errors
type LogError struct {
	Level     LogLevel
	Message   string
	Component string
	Caller    string
}

func (e *LogError) Error() string {
	return fmt.Sprintf("log error in %s [%s]: %s (caller: %s)", e.Component, e.Level, e.Message, e.Caller)
}

// Predefined errors
var (
	ErrStackOverflow        = &StackError{Operation: "push", Message: "stack overflow"}
	ErrStackUnderflow       = &StackError{Operation: "pop", Message: "stack underflow"}
	ErrStackPeekOutOfBounds = &StackError{Operation: "peek", Message: "out of bounds"}
	ErrStackDupOutOfBounds  = &StackError{Operation: "dup", Message: "out of bounds"}
	ErrStackSwapOutOfBounds = &StackError{Operation: "swap", Message: "out of bounds"}

	ErrInvalidOpcode      = &EVMError{Code: 1, Message: "invalid opcode"}
	ErrInvalidJump        = &EVMError{Code: 2, Message: "invalid jump destination"}
	ErrCodeCopyOutOfRange = &EVMError{Code: 3, Message: "CODECOPY out of range"}
	ErrPushOutOfRange     = &EVMError{Code: 4, Message: "invalid PUSH: out of range"}

	ErrMissingBinOrArtifact = &ConfigError{Field: "input", Message: "-bin or -artifact flag is required"}
	ErrMissingBlockNumber   = &ConfigError{Field: "block", Message: "-block must be provided"}
	ErrMissingStartEnd      = &ConfigError{Field: "range", Message: "both -start and -end must be provided"}
	ErrInvalidRange         = &ConfigError{Field: "range", Message: "-start must be less than or equal to -end"}
)

// Helper functions to create errors with context
func NewStackPeekError(n, length int) error {
	return &StackError{
		Operation: "peek",
		Message:   fmt.Sprintf("out of bounds: %d >= %d", n, length),
		StackSize: length,
		Requested: n,
	}
}

func NewStackDupError(n int) error {
	return &StackError{
		Operation: "dup",
		Message:   fmt.Sprintf("out of bounds: n=%d", n),
		Requested: n,
	}
}

func NewStackSwapError(n int) error {
	return &StackError{
		Operation: "swap",
		Message:   fmt.Sprintf("out of bounds: n=%d", n),
		Requested: n,
	}
}

func NewInvalidOpcodeError(op byte, pc uint64) error {
	return &EVMError{
		Code:    1,
		Message: fmt.Sprintf("unsupported opcode 0x%02x", op),
		PC:      pc,
		Opcode:  op,
	}
}

func NewInvalidJumpError(target uint64, pc uint64) error {
	return &EVMError{
		Code:    2,
		Message: fmt.Sprintf("invalid jump destination 0x%x", target),
		PC:      pc,
	}
}

func NewInvalidOpcodeExecutionError(op byte, pc uint64) error {
	return &EVMError{
		Code:    1,
		Message: fmt.Sprintf("execution hit INVALID opcode: 0x%02x", op),
		PC:      pc,
		Opcode:  op,
	}
}

func NewRPCError(method, message, requestID string, code int) error {
	return &RPCError{
		Method:    method,
		Message:   message,
		RequestID: requestID,
		Code:      code,
	}
}

func NewLogError(level LogLevel, message, component string) error {
	_, file, line, _ := runtime.Caller(1)
	caller := fmt.Sprintf("%s:%d", file, line)

	return &LogError{
		Level:     level,
		Message:   message,
		Component: component,
		Caller:    caller,
	}
}

// IsEVMError checks if an error is an EVM error
func IsEVMError(err error) bool {
	_, ok := err.(*EVMError)
	return ok
}

// IsStackError checks if an error is a stack error
func IsStackError(err error) bool {
	_, ok := err.(*StackError)
	return ok
}

// IsConfigError checks if an error is a config error
func IsConfigError(err error) bool {
	_, ok := err.(*ConfigError)
	return ok
}

// IsRPCError checks if an error is an RPC error
func IsRPCError(err error) bool {
	_, ok := err.(*RPCError)
	return ok
}

// IsLogError checks if an error is a log error
func IsLogError(err error) bool {
	_, ok := err.(*LogError)
	return ok
}
