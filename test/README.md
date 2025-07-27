# EchoEVM 测试目录

这个目录包含了 EchoEVM 项目的所有测试相关文件，包括智能合约、测试脚本和工具。

## 目录结构

```
test/
├── bins/                   # 二进制合约文件
│   ├── Add.sol
│   ├── Multiply.sol
│   ├── Sum.sol
│   └── build/              # 编译后的二进制文件
├── contract/               # Hardhat 合约开发环境
│   ├── contracts/          # Solidity 源码
│   ├── artifacts/          # 编译产物
│   ├── test/              # Hardhat 单元测试
│   └── hardhat.config.ts   # Hardhat 配置
├── scripts/                # 测试执行脚本
│   ├── basic.sh            # 基础测试
│   ├── advanced.sh         # 高级测试
│   └── run_all.sh          # 运行所有测试
├── config/                 # 测试配置文件
│   ├── test_cases.toml     # 测试用例定义
│   └── environments.toml   # 环境配置
├── utils/                  # 测试工具函数
│   ├── helpers.sh          # 通用辅助函数
│   └── contract_utils.sh   # 合约相关工具
├── docs/                   # 测试文档
│   ├── TESTING_GUIDE.md    # 测试指南
│   └── examples/           # 测试示例
└── reports/                # 测试报告
    ├── latest/             # 最新测试结果
    └── history/            # 历史测试记录
```

## 使用方法

### 运行测试
```bash
# 运行所有测试
cd test/scripts && ./run_all.sh

# 运行基础测试
cd test/scripts && ./basic.sh

# 运行高级测试
cd test/scripts && ./advanced.sh
```

### 添加新测试
1. 在 `config/test_cases.toml` 中定义新的测试用例
2. 如需要，在 `contracts/` 中添加新的合约
3. 运行测试验证

### 查看测试结果
测试结果保存在 `reports/` 目录中，包括：
- 执行日志
- 性能数据
- 错误报告

## 开发工具

- **合约开发**: 使用 `contract/` 目录下的 Hardhat 环境
- **二进制合约**: 使用 `bins/` 目录下的预编译合约
- **测试工具**: 使用 `utils/` 目录下的辅助函数

## 注意事项

1. 确保安装了必要的依赖（Go、Node.js、jq）
2. 运行测试前先编译项目：`make build`
3. 测试脚本需要执行权限：`chmod +x test/scripts/*.sh`
