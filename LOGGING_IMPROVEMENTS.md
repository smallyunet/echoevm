# EchoEVM 日志系统改进

## 概述

本次改进对 EchoEVM 项目的日志系统进行了全面升级，提高了项目的可靠性和可维护性。

## 主要改进

### 1. 统一的日志系统
- 创建了 `internal/logger` 包，提供结构化的日志记录
- 支持多种输出格式（控制台、JSON）
- 支持多种输出目标（stdout、stderr、文件）
- 统一的日志级别管理

### 2. 增强的错误处理
- 扩展了 `internal/errors` 包，添加了更多错误类型
- 结构化的错误信息，包含上下文数据
- 支持错误类型检查和分类

### 3. 结构化的日志记录
- 所有日志都包含组件、版本等上下文信息
- EVM 执行日志包含程序计数器、操作码、栈状态等详细信息
- RPC 请求日志包含方法名、请求ID、响应时间等
- 合约执行日志包含地址、函数名、gas 使用量等

### 4. 日志级别标准化
- **TRACE**: 详细的执行跟踪（EVM 操作码执行、栈操作等）
- **DEBUG**: 调试信息（字节码反汇编、配置加载等）
- **INFO**: 一般操作信息（应用启动/关闭、RPC 服务器状态等）
- **WARN**: 警告条件（缺少合约代码、性能警告等）
- **ERROR**: 错误条件（EVM 执行错误、栈错误、RPC 错误等）
- **FATAL**: 严重错误（启动失败、关键配置错误等）

## 使用示例

### 基本日志记录
```go
logger.Info().Msg("Application starting")
logger.Error().Err(err).Msg("Failed to process transaction")
```

### EVM 执行日志
```go
logger.EVMExecution(pc, opcode, stack, gas)
logger.EVMError(pc, opcode, err, stack)
```

### RPC 请求日志
```go
logger.RPCRequest(method, requestID, params)
logger.RPCResponse(method, requestID, result, duration)
logger.RPCError(method, requestID, err, duration)
```

### 合约执行日志
```go
logger.ContractExecution(address, function, input, gas)
logger.ContractResult(address, output, gasUsed, success)
```

## 配置

### 环境变量
```bash
export ECHOEVM_LOG_LEVEL=debug
export ECHOEVM_LOG_FORMAT=json
export ECHOEVM_LOG_OUTPUT=/var/log/echoevm.log
```

### 命令行参数
```bash
./echoevm --log-level=debug --log-format=json --log-file=echoevm.log
```

## 日志格式示例

### 控制台格式
```
2024-01-15T10:30:45.123Z INF EVM execution step component=echoevm pc=0x0000 opcode=0x60 opcode_name=PUSH1 stack_size=0
2024-01-15T10:30:45.124Z INF RETURN operation executed component=echoevm offset=0x00 size=0x20 return_data_hex=0x1234... return_data_size=32
```

### JSON 格式
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

## 监控和告警

### 关键指标
- 错误率（按组件）
- RPC 调用响应时间
- EVM 执行成功率
- 内存和栈使用模式

### 告警规则
- 错误率 > 5%（5分钟内）
- RPC 响应时间 > 1秒
- EVM 执行失败率 > 10%（1分钟内）
- 内存使用率 > 80%

## 最佳实践

1. **使用适当的日志级别**
   - 生产环境使用 INFO 或更高级别
   - 开发环境可以使用 DEBUG 或 TRACE
   - 只在关键错误时使用 FATAL

2. **包含相关上下文**
   - 始终包含组件和版本信息
   - 包含相关ID（request_id、tx_hash等）
   - 包含性能指标（如适用）

3. **结构化日志**
   - 使用结构化字段而不是字符串插值
   - 跨组件使用一致的字段名
   - 包含人可读和机器可读的格式

4. **性能考虑**
   - 生产环境中谨慎使用 TRACE 级别
   - 考虑文件输出的日志轮转
   - 对高容量操作使用适当的采样

5. **安全性**
   - 永远不要记录敏感数据（私钥、密码）
   - 小心处理用户输入
   - 考虑外部系统的日志清理

## 迁移指南

### 从旧日志系统迁移
1. 将 `fmt.Printf` 替换为结构化日志
2. 将 `log.Println` 替换为适当的日志级别
3. 为现有日志消息添加上下文字段
4. 更新错误处理以包含结构化错误信息

### 示例迁移
```go
// 旧方式
fmt.Printf("Executing opcode 0x%02x at PC 0x%04x\n", opcode, pc)

// 新方式
logger.Trace().
    Uint8("opcode", opcode).
    Str("opcode_hex", fmt.Sprintf("0x%02x", opcode)).
    Uint64("pc", pc).
    Str("pc_hex", fmt.Sprintf("0x%04x", pc)).
    Msg("Executing opcode")
```

## 故障排除

### 常见问题
1. **日志量过大**: 提高日志级别或实现采样
2. **缺少上下文**: 确保所有日志调用都包含相关字段
3. **性能影响**: 使用适当的日志级别并考虑异步日志
4. **存储问题**: 实现日志轮转和压缩

### 调试命令
```bash
# 设置调试级别以获取详细日志
export ECHOEVM_LOG_LEVEL=debug

# 使用 JSON 格式进行日志聚合
export ECHOEVM_LOG_FORMAT=json

# 将日志重定向到文件
export ECHOEVM_LOG_OUTPUT=/var/log/echoevm.log
```

## 总结

通过这些改进，EchoEVM 项目现在拥有了一个强大、可靠、易于使用的日志系统。该系统提供了：

- 统一的日志接口
- 结构化的日志数据
- 灵活的配置选项
- 全面的错误处理
- 详细的执行跟踪
- 性能监控支持

这些改进将大大提高项目的可维护性、调试能力和生产环境的可靠性。
