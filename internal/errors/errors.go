package errors

import (
	"fmt"
)

// EVM execution errors
type EVMError struct {
	Code    int
	Message string
}

func (e *EVMError) Error() string {
	return fmt.Sprintf("EVM error %d: %s", e.Code, e.Message)
}

// Stack errors
type StackError struct {
	Operation string
	Message   string
}

func (e *StackError) Error() string {
	return fmt.Sprintf("stack %s error: %s", e.Operation, e.Message)
}

// Configuration errors
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error for %s: %s", e.Field, e.Message)
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
	}
}

func NewStackDupError(n int) error {
	return &StackError{
		Operation: "dup",
		Message:   fmt.Sprintf("out of bounds: n=%d", n),
	}
}

func NewStackSwapError(n int) error {
	return &StackError{
		Operation: "swap",
		Message:   fmt.Sprintf("out of bounds: n=%d", n),
	}
}

func NewInvalidOpcodeError(op byte) error {
	return &EVMError{
		Code:    1,
		Message: fmt.Sprintf("unsupported opcode 0x%02x", op),
	}
}

func NewInvalidJumpError(target uint64) error {
	return &EVMError{
		Code:    2,
		Message: fmt.Sprintf("invalid jump destination 0x%x", target),
	}
}

func NewInvalidOpcodeExecutionError(op byte) error {
	return &EVMError{
		Code:    1,
		Message: fmt.Sprintf("execution hit INVALID opcode: 0x%02x", op),
	}
}
