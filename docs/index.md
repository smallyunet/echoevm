# echoevm

**EchoEVM** is a minimal, pedagogical Ethereum Virtual Machine (EVM) implementation written in Go. It focuses on transparent bytecode execution, traceability, and ease of experimentation rather than production consensus or networking features.

---


## 🆕 What's New in v0.0.17

- **Web Debugger Run Control**: Trigger traces directly from the UI via the new Run button.
- **Web Debugger Origin Allowlist**: Configure allowed WebSocket origins with `ECHOEVM_WEB_ALLOWED_ORIGINS`.
- **Docs & Version Alignment**: Updated docs and tests to reflect the current release.

See [ROADMAP.md](https://github.com/smallyunet/echoevm/blob/main/ROADMAP.md) for the complete version history.

---

## ✨ Features

| Category | Features |
|----------|----------|
| **Execution** | Constructor deployment, runtime calls, bytecode disassembly |
| **ABI Support** | Function selector encoding, primitives, arrays, bytes types |
| **Tracing** | JSON structured per-opcode tracing with pre/post state |
| **Gas Metering** | EIP-2929 compatible dynamic gas calculations |
| **EIP Support** | EIP-1153 (Transient Storage), EIP-5656 (MCOPY) |
| **Precompiles** | ECRECOVER..BLAKE2F (0x01-0x09) |
| **Testing** | Unit tests, integration tests, Ethereum compliance tests |
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
make test-compliance  # Run Ethereum compliance tests
make setup-tests      # Download test fixtures (~100MB)
```

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

See [Configuration](guides/configuration.md) and [Logging Guide](guides/logging.md) for details.

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

See **[ROADMAP.md](https://github.com/smallyunet/echoevm/blob/main/ROADMAP.md)** for the complete development roadmap.

**Upcoming:**
- Tuple and nested array ABI support
- Fork-specific opcode behavior (Cancun)
- Improved compliance test coverage
- Web-based debugger UI

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

This project is open source under the MIT License. See [LICENSE](https://github.com/smallyunet/echoevm/blob/main/LICENSE) for details.

---

<p align="center">
  <i>If you're using EchoEVM in research, experiments, or education, a citation or link back is appreciated.</i>
</p>
