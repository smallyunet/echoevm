# EchoEVM 硬编码问题修复报告

## 概述

本报告详细说明了在 EchoEVM 项目中发现的硬编码问题以及相应的修复方案。

## 发现的硬编码问题

### 1. 网络和端点配置
- **文件**: `cmd/echoevm/flags.go`
- **问题**: 
  - 硬编码的 RPC 端点: `"localhost:8545"`
  - 硬编码的以太坊 RPC: `"https://cloudflare-eth.com"`
  - 硬编码的日志级别: `"info"`
  - 硬编码的执行模式: `"full"`

### 2. EVM 常量
- **文件**: `internal/evm/core/stack.go`
- **问题**: 硬编码的栈大小限制: `1024`

### 3. RPC API 配置
- **文件**: `internal/rpc/eth_api.go`
- **问题**:
  - 硬编码的 gas 限制: `15000000`
  - 硬编码的日志 bloom 大小: `256`
  - 硬编码的时间戳: `0`

### 4. RPC 服务器配置
- **文件**: `internal/rpc/server.go`
- **问题**:
  - 硬编码的 API 命名空间: `"eth"`
  - 硬编码的 API 版本: `"1.0"`
  - 硬编码的 API 公开标志: `true`

### 5. 日志配置
- **文件**: `internal/logger/logger.go`
- **问题**:
  - 硬编码的日志级别: `"info"`
  - 硬编码的日志格式: `"console"`
  - 硬编码的日志输出: `"stdout"`
  - 硬编码的组件名称: `"echoevm"`
  - 硬编码的版本: `"1.0.0"`
  - 硬编码的文件权限: `0666`

## 修复方案

### 1. 创建配置常量文件
**文件**: `internal/config/constants.go`

创建了一个集中的配置常量文件，包含所有可配置的默认值：

```go
// EVM Constants
const (
    StackLimit = 1024
    DefaultGasLimit = 15000000
    DefaultBlockGasLimit = 15000000
    LogsBloomSize = 256
    DefaultTimestamp = 1640995200
)

// RPC Constants
const (
    DefaultRPCEndpoint = "localhost:8545"
    DefaultRPCTimeout = 30 * time.Second
    // ... 更多常量
)
```

### 2. 创建环境变量配置系统
**文件**: `internal/config/env.go`

实现了环境变量覆盖机制，支持运行时配置：

```go
// Environment variable names
const (
    EnvRPCEndpoint = "ECHOEVM_RPC_ENDPOINT"
    EnvLogLevel = "ECHOEVM_LOG_LEVEL"
    // ... 更多环境变量
)

// RuntimeConfig holds runtime configuration
type RuntimeConfig struct {
    RPCEndpoint string
    LogLevel    string
    // ... 更多配置字段
}
```

### 3. 更新现有文件使用配置常量

#### cmd/echoevm/flags.go
- 替换硬编码的默认值为配置常量
- 添加配置包导入

#### internal/evm/core/stack.go
- 移除硬编码的 `StackLimit` 常量
- 使用 `config.StackLimit`

#### internal/rpc/eth_api.go
- 使用 `config.DefaultBlockGasLimit` 替换硬编码的 gas 限制
- 使用 `config.LogsBloomSize` 替换硬编码的 bloom 大小
- 使用 `config.DefaultTimestamp` 替换硬编码的时间戳

#### internal/rpc/server.go
- 使用配置常量替换硬编码的 API 配置

#### internal/evm/core/stack_test.go
- 更新测试文件以使用新的配置常量

## 配置优先级

修复后的配置系统支持以下优先级（从高到低）：

1. **命令行参数** - 最高优先级
2. **环境变量** - 中等优先级
3. **配置常量** - 最低优先级

## 环境变量支持

现在支持以下环境变量来覆盖默认配置：

### RPC 配置
- `ECHOEVM_RPC_ENDPOINT`: HTTP RPC 端点地址
- `ECHOEVM_ETHEREUM_RPC`: 以太坊 RPC 端点
- `ECHOEVM_RPC_TIMEOUT`: RPC 调用超时

### 日志配置
- `ECHOEVM_LOG_LEVEL`: 日志级别
- `ECHOEVM_LOG_FORMAT`: 日志格式
- `ECHOEVM_LOG_OUTPUT`: 日志输出

### EVM 配置
- `ECHOEVM_GAS_LIMIT`: 默认 gas 限制
- `ECHOEVM_BLOCK_GAS_LIMIT`: 默认区块 gas 限制
- `ECHOEVM_CHAIN_ID`: 默认链 ID

### API 配置
- `ECHOEVM_API_NAMESPACE`: 默认 API 命名空间
- `ECHOEVM_API_VERSION`: 默认 API 版本
- `ECHOEVM_API_PUBLIC`: 默认 API 公开标志

## 使用示例

### 设置 RPC 端点
```bash
export ECHOEVM_RPC_ENDPOINT="0.0.0.0:8545"
./echoevm serve
```

### 设置日志级别
```bash
export ECHOEVM_LOG_LEVEL="debug"
./echoevm run -bin contract.bin
```

### 设置 gas 限制
```bash
export ECHOEVM_GAS_LIMIT="30000000"
./echoevm run -bin contract.bin
```

## 测试验证

所有修复都通过了以下测试：

- ✅ 项目编译成功
- ✅ 核心 EVM 功能测试通过
- ✅ 命令行参数解析测试通过
- ✅ RPC 功能测试通过

## 文档更新

创建了详细的配置指南文档：
- `docs/CONFIGURATION.md`: 完整的配置使用指南

## 最佳实践建议

1. **生产环境部署**: 使用环境变量而不是硬编码值
2. **日志管理**: 生产环境使用 JSON 格式便于日志聚合
3. **安全性**: 注意文件权限设置，生产环境可能需要更严格的权限
4. **性能调优**: 根据具体使用场景调整 gas 限制和超时设置
5. **监控**: 使用适当的日志级别平衡调试信息和性能

## 总结

通过这次修复，EchoEVM 项目现在具有：

- ✅ 集中的配置管理
- ✅ 环境变量支持
- ✅ 灵活的配置覆盖机制
- ✅ 更好的可维护性
- ✅ 生产环境就绪的配置系统

这些改进使得 EchoEVM 更适合在不同环境中部署，并且更容易进行配置管理和维护。
