# echoevm

[![Go Version](https://img.shields.io/badge/go-1.23.2+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Release](https://img.shields.io/github/v/release/smallyunet/echoevm?style=flat&color=blue)](https://github.com/smallyunet/echoevm/releases)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen?style=flat)]()

**EchoEVM** is a minimal, pedagogical Ethereum Virtual Machine (EVM) implementation written in Go. It focuses on transparent bytecode execution, traceability, and ease of experimentation rather than production consensus or networking features.

---

## 📑 Table of Contents

- [What's New in v0.0.21](#-whats-new-in-v0021)
- [Features](#-features)
- [Requirements](#-requirements)
- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [CLI Commands](#-cli-commands)
- [ABI Encoding](#-abi-encoding)
- [Testing](#-testing)
- [Architecture](#-architecture)
- [Configuration](#%EF%B8%8F-configuration)
- [Roadmap](#-roadmap)
- [Contributing](#-contributing)
- [License](#-license)

---

## 🆕 What's New in v0.0.21

- **Geth Differential Conformance**: 17 Cancun vectors compare return data, gas used, halt class, and persistent storage against go-ethereum across eight behavior categories.
- **Expanded Official Baseline**: Pinned Cancun ADD, MUL, and SUB fixtures increase the official baseline from 3 to 9 cases.
- **Non-Shrinking Baselines**: CI fails if official fixtures, differential vectors, required metadata, or required categories disappear.
- **Visible Conformance Reports**: CI prints official and differential case counts by fork and category with an explicit zero-skip contract.
- **Complete EIP-152 Precompile**: BLAKE2F (0x09) now validates, charges, and executes the BLAKE2b compression function.

### Previous v0.0.20

- **Correct Transaction Semantics**: Prechecks no longer mutate state, exceptional halts consume gas and return errors, and REVERT remains distinguishable from execution errors.
- **Transaction Isolation**: Refunds, access lists, transient storage, journals, and original storage snapshots reset between transactions.
- **Top-Level Precompiles**: Transactions addressed directly to precompiled contracts now execute through the native implementation.
- **Reliable Compliance Baseline**: Three pinned official Cancun vectors run offline and the suite fails instead of silently passing with zero fixtures.
- **Machine-Detectable CLI Failures**: Transaction JSON output is preserved while exceptional halts and REVERT return a non-zero exit code.

### Previous v0.0.19

- **Consistent Execution**: `run`, debug tracing, JSON tracing, and the Web Debugger now share one gas-aware interpreter loop.
- **Reliable CLI Commands**: `run`, `deploy`, `call`, `trace`, `repl`, and `web` use a working default gas budget and return execution errors consistently.
- **Web Debugger Restored**: Restored the missing `web` command and fixed WebSocket trace message framing.
- **Trie Stability**: Fixed prefix-key insertion panics in the Merkle Patricia Trie.

### Previous v0.0.18

- **Merkle Patricia Trie (MPT)**: Full implementation of the Ethereum state trie (`internal/trie`), satisfying the Yellow Paper structure.
- **Trie-backed Reads**: StateDB can lazily load accounts and storage from `TrieStateBackend`; committing modified state roots is not yet supported.
- **Compliance Baseline**: A small, pinned subset of official Ethereum execution vectors runs in the normal test suite.
- **RLP & Compact Encoding**: Custom encoding implementations for MPT nodes.

See [ROADMAP.md](ROADMAP.md) for the complete version history.

---

## ✨ Features

| Category | Features |
|----------|----------|
| **Execution** | Constructor deployment, runtime calls, bytecode disassembly |
| **State Management** | **Merkle Patricia Trie**, lazy trie-backed reads, in-memory journaling |
| **ABI Support** | Function selector encoding, primitives, arrays, bytes types |
| **Tracing** | JSON structured per-opcode tracing with pre/post state |
| **Gas Metering** | EIP-2929 compatible dynamic gas calculations |
| **EIP Support** | EIP-1153 (Transient Storage), EIP-5656 (MCOPY) |
| **Precompiles** | ECRECOVER..BLAKE2F (0x01-0x09) |
| **Testing** | Unit, integration, E2E, pinned official fixtures, geth differential conformance |
| **Logging** | Zerolog-based structured logging (plain/JSON output) |

---

## ✅ Requirements

- Go 1.23.2+
- (Optional) `solc` for compiling `.sol` files directly

---

## 🔧 Installation

**From source:**

```bash
go install github.com/smallyunet/echoevm/cmd/echoevm@latest
```

**Clone and build:**

```bash
git clone https://github.com/smallyunet/echoevm.git
cd echoevm
make build
make install   # install into GOPATH/bin
```

**Verify:**

```bash
echoevm --help
```

---

## 🚀 Quick Start

### Run bytecode directly

```bash
# Simple arithmetic: PUSH1 1 PUSH1 2 ADD
echoevm run 6001600201

# With debug trace
echoevm run --debug 6001600201
```

### Deploy and call a contract

```bash
# Deploy constructor bytecode
echoevm deploy -a ./artifacts/Add.json --print

# Call a function with ABI encoding
echoevm call -a ./artifacts/Add.json -f add(uint256,uint256) -A 2,40

# Generate execution trace
echoevm trace -a ./artifacts/Add.json -f add(uint256,uint256) -A 7,9 --full | jq .
```

### Interactive REPL

```bash
echoevm repl
echoevm repl
# Type opcodes: PUSH1 10 PUSH1 20 ADD
```

### Web Debugger

```bash
# Start the web debugger
echoevm web --code "6003600401"
# Then open http://localhost:8080
# Click "Run Trace" in the UI to start execution.
```

---

## 🖥 CLI Commands

| Command | Description |
|---------|-------------|
| `run` | Execute raw bytecode with optional debug tracing |
| `deploy` | Run constructor and extract runtime bytecode |
| `call` | Execute runtime bytecode with ABI encoding |
| `trace` | JSON line trace of opcode execution |
| `disasm` | Disassemble bytecode to human-readable opcodes |
| `repl` | Interactive EVM shell |
| `web` | Browser-based visual debugger |
| `version` | Display build metadata |

### Global Flags

```
--log-level, -L   Log level (trace|debug|info|warn|error)
--output, -o      Output format (plain|json)
--config, -c      Config file path (reserved)
--rpc-url         Ethereum RPC endpoint
```

### Command Examples

<details>
<summary><b>deploy</b> - Execute constructor bytecode</summary>

```bash
echoevm deploy -a ./artifacts/Add.json --print
echoevm deploy -b ./constructor.bin --out-file runtime.bin
```
</details>

<details>
<summary><b>disasm</b> - Disassemble bytecode</summary>

```bash
# From hex
echoevm disasm 6001600201
# Output:
# 0000: PUSH1 01
# 0002: PUSH1 02
# 0004: ADD

# From artifact
echoevm disasm -a ./artifacts/Add.json --runtime

# JSON output
echoevm disasm -o json 6001600201
```
</details>

<details>
<summary><b>call</b> - Execute runtime bytecode</summary>

```bash
# With ABI encoding
echoevm call -a ./artifacts/Add.json -f add(uint256,uint256) -A 2,40

# With raw calldata
echoevm call -r ./runtime.bin -d 771602f70000...
```
</details>

<details>
<summary><b>trace</b> - Execution trace</summary>

```bash
# First 40 steps
echoevm trace -a ./artifacts/Add.json -f add(uint256,uint256) -A 1,2 --limit 40 | jq .

# Full pre/post state
echoevm trace -a ./artifacts/Loops.json -f forLoop(uint256) -A 5 --full | jq .
```
</details>

---

## 📦 ABI Encoding

Supported types for `--function/--args` encoding:

| Type | Examples |
|------|----------|
| Integers | `uint8`, `uint256`, `int128`, etc. |
| Boolean | `true`, `false` |
| Address | `0x742d35Cc...` (40 hex chars) |
| String | UTF-8 dynamic strings |
| Bytes | `bytes` (dynamic), `bytes32` (fixed) |
| Arrays | `uint256[]`, `address[]` |

**Array syntax:**

```bash
echoevm call -a ./artifacts/Sum.json -f sum(uint256[]) -A "[1;2;3;4;5]"
echoevm call -a ./artifacts/Multi.json -f send(address[]) -A "[0xabc...;0xdef...]"
```

> **Note:** Tuples and nested arrays are supported.

---

## 🧪 Testing

```bash
make test             # Run all tests (unit, integration, compliance)
make test-unit        # Run Go unit tests
make test-integration # Run integration tests
make test-compliance  # Run the pinned Ethereum compliance baseline
make test-differential # Compare Cancun behavior with go-ethereum
make test-conformance # Run both conformance layers with summary output
```

The v0.0.21 baseline contains 9 pinned official Cancun cases and 17 geth
differential vectors across arithmetic, bitwise, control, crypto, environment,
fault, memory, and storage. Both suites fail on missing metadata, shrinking
case counts, missing required categories, or skipped execution.

---

## 🏗 Architecture

```
echoevm/
├── cmd/echoevm/     # CLI commands (deploy, call, trace, etc.)
├── internal/
│   ├── evm/
│   │   ├── core/    # Stack, memory, opcode table
│   │   └── vm/      # Interpreter, opcode implementations
│   ├── config/      # Constants, environment variables
│   ├── logger/      # Zerolog wrapper
│   └── errors/      # Error definitions
├── tests/           # Integration and compliance tests
└── docs/            # Documentation
```

### Supported Opcode Categories

Arithmetic, Bitwise, Comparison, Stack, Memory, Storage, Control Flow, Environment, Call/Return/Revert, Logging, System.

---

## ⚙️ Configuration

See [docs/guides/configuration.md](docs/guides/configuration.md) and [docs/guides/logging.md](docs/guides/logging.md) for details.

**Environment variables:**

```bash
export ECHOEVM_LOG_LEVEL=debug
export ECHOEVM_GAS_LIMIT=30000000
export ECHOEVM_ETHEREUM_RPC="https://mainnet.infura.io/v3/<key>"
```

---

## 🚦 Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Successful execution |
| 1 | Execution reverted (REVERT) |
| 2 | Invalid arguments / configuration error |

---

## 🗺 Roadmap

See **[ROADMAP.md](ROADMAP.md)** for the complete development roadmap.

**Upcoming:**
- Tuple and nested array ABI support
- Fork-specific opcode behavior (Cancun)
- Improved compliance test coverage
- [x] Web-based debugger UI

---

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`feat/<topic>`)
3. Add/update tests
4. Run `make test` and ensure build is clean
5. Open a PR with clear description

Issues and discussions for roadmap ideas are welcome!

---

## 📄 License

This project is open source under the MIT License. See [LICENSE](LICENSE) for details.

---

<p align="center">
  <i>If you're using EchoEVM in research, experiments, or education, a citation or link back is appreciated.</i>
</p>
