package config

import (
	"os"
	"strconv"
	"time"
)

// Environment variable names
const (
	EnvRPCEndpoint   = "ECHOEVM_RPC_ENDPOINT"
	EnvLogLevel      = "ECHOEVM_LOG_LEVEL"
	EnvLogFormat     = "ECHOEVM_LOG_FORMAT"
	EnvLogOutput     = "ECHOEVM_LOG_OUTPUT"
	EnvEthereumRPC   = "ECHOEVM_ETHEREUM_RPC"
	EnvGasLimit      = "ECHOEVM_GAS_LIMIT"
	EnvBlockGasLimit = "ECHOEVM_BLOCK_GAS_LIMIT"
	EnvChainID       = "ECHOEVM_CHAIN_ID"
	EnvExecutionMode = "ECHOEVM_EXECUTION_MODE"
	EnvAPINamespace  = "ECHOEVM_API_NAMESPACE"
	EnvAPIVersion    = "ECHOEVM_API_VERSION"
	EnvAPIPublic     = "ECHOEVM_API_PUBLIC"
)

// GetStringEnv returns environment variable value or default
func GetStringEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetIntEnv returns environment variable value as int or default
func GetIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetUint64Env returns environment variable value as uint64 or default
func GetUint64Env(key string, defaultValue uint64) uint64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseUint(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetBoolEnv returns environment variable value as bool or default
func GetBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetDurationEnv returns environment variable value as time.Duration or default
func GetDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// RuntimeConfig holds runtime configuration that can be overridden by environment variables
type RuntimeConfig struct {
	RPCEndpoint     string
	LogLevel        string
	LogFormat       string
	LogOutput       string
	EthereumRPC     string
	GasLimit        uint64
	BlockGasLimit   uint64
	ChainID         uint64
	ExecutionMode   string
	APINamespace    string
	APIVersion      string
	APIPublic       bool
	RPCTimeout      time.Duration
	RPCReadTimeout  time.Duration
	RPCWriteTimeout time.Duration
	RPCIdleTimeout  time.Duration
}

// LoadRuntimeConfig loads configuration from environment variables with defaults
func LoadRuntimeConfig() *RuntimeConfig {
	return &RuntimeConfig{
		RPCEndpoint:     GetStringEnv(EnvRPCEndpoint, DefaultRPCEndpoint),
		LogLevel:        GetStringEnv(EnvLogLevel, DefaultLogLevel),
		LogFormat:       GetStringEnv(EnvLogFormat, DefaultLogFormat),
		LogOutput:       GetStringEnv(EnvLogOutput, DefaultLogOutput),
		EthereumRPC:     GetStringEnv(EnvEthereumRPC, DefaultEthereumRPC),
		GasLimit:        GetUint64Env(EnvGasLimit, DefaultGasLimit),
		BlockGasLimit:   GetUint64Env(EnvBlockGasLimit, DefaultBlockGasLimit),
		ChainID:         GetUint64Env(EnvChainID, DefaultChainID),
		ExecutionMode:   GetStringEnv(EnvExecutionMode, DefaultExecutionMode),
		APINamespace:    GetStringEnv(EnvAPINamespace, DefaultAPINamespace),
		APIVersion:      GetStringEnv(EnvAPIVersion, DefaultAPIVersion),
		APIPublic:       GetBoolEnv(EnvAPIPublic, DefaultAPIPublic),
		RPCTimeout:      GetDurationEnv("ECHOEVM_RPC_TIMEOUT", DefaultRPCTimeout),
		RPCReadTimeout:  GetDurationEnv("ECHOEVM_RPC_READ_TIMEOUT", DefaultRPCReadTimeout),
		RPCWriteTimeout: GetDurationEnv("ECHOEVM_RPC_WRITE_TIMEOUT", DefaultRPCWriteTimeout),
		RPCIdleTimeout:  GetDurationEnv("ECHOEVM_RPC_IDLE_TIMEOUT", DefaultRPCIdleTimeout),
	}
}

// Global runtime configuration instance
var globalRuntimeConfig *RuntimeConfig

// GetRuntimeConfig returns the global runtime configuration
func GetRuntimeConfig() *RuntimeConfig {
	if globalRuntimeConfig == nil {
		globalRuntimeConfig = LoadRuntimeConfig()
	}
	return globalRuntimeConfig
}

// SetRuntimeConfig sets the global runtime configuration
func SetRuntimeConfig(config *RuntimeConfig) {
	globalRuntimeConfig = config
}
