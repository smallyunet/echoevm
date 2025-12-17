# echoevm

EchoEVM is a minimal, pedagogical Ethereum Virtual Machine (EVM) implementation written in Go. It focuses on transparent bytecode execution, traceability, and ease of experimentation rather than production consensus or networking features.

## ✨ Features

- **Constructor Deployment**: Execute constructor bytecode and extract emitted runtime code (`deploy`).
- **Runtime Calls**: Execute deployed / runtime bytecode with ABI encoded calldata (`call`).
- **Bytecode Disassembly**: Human-readable opcode disassembly from hex or artifacts (`disasm`).
- **ABI Convenience**: Lightweight ABI function selector & argument encoding for common primitive types.
- **Execution Tracing**: JSON structured per-opcode tracing with optional pre/post state (`trace`).
- **Gas Accounting**: EIP-2929 compatible gas metering with dynamic cost calculations.
- **Transient Storage**: Full support for EIP-1153 (TLOAD/TSTORE).
- **Memory Copy**: Efficient memory copying with MCOPY (EIP-5656).
- **Deterministic Core**: Small, auditable interpreter with clear stack & memory semantics.
- **Testing Suite**: Go unit tests covering opcodes, stack, memory, control and ABI paths.
- **Structured Logging**: Zerolog based, selectable output format (plain | json) and adjustable log level.

Planned / in-progress (roadmap): expanded ABI types support for tuples.

## ✅ Requirements

- Go 1.23.2+
- (Optional) `solc` if compiling standalone `.sol` files directly.

## 🔧 Installation

Install from source:

```bash
go install github.com/smallyunet/echoevm/cmd/echoevm@latest
```

Or clone and build:

```bash
git clone https://github.com/smallyunet/echoevm.git
cd echoevm
make build
make install   # install into GOPATH/bin
```

Verify:

```bash
echoevm --help
```

## 🖥 CLI Overview

Global flags (apply to all subcommands):

```
--log-level, -L   Log level (trace|debug|info|warn|error) (default: info)
--output, -o      Output formatting for command responses (plain|json) (default: plain)
--config, -c      Optional config file path (reserved for future use)
--rpc-url         Default Ethereum RPC endpoint (used by planned commands)
```

### Subcommands

#### 1. deploy
Execute constructor bytecode (from a `.bin` file or Hardhat artifact) and emit the resulting runtime code.

Flags:
```
--bin, -b         Constructor .bin file path
--artifact, -a    Hardhat artifact JSON containing "bytecode"
--out-file        Write runtime hex (no 0x prefix) to a file
--print           Also print runtime hex to stdout (auto if no --out-file)
```

Example:
```bash
echoevm deploy -a ./artifacts/Add.json --print
```

#### 2. disasm
Disassemble EVM bytecode into human-readable opcode sequences.

Flags:
```
--bin, -b         Path to .bin file containing bytecode
--artifact, -a    Hardhat artifact JSON
--runtime, -r     Use deployedBytecode from artifact (default: constructor)
```

Example (from hex):
```bash
echoevm disasm 6001600201
# Output:
# 0000: PUSH1 01
# 0002: PUSH1 02
# 0004: ADD
```

Example (from artifact):
```bash
echoevm disasm -a ./artifacts/Add.json --runtime
```

Example (JSON output):
```bash
echoevm disasm -o json 6001600201
```

#### 3. call
Execute runtime (deployed) bytecode and optionally ABI-encode function calls.

Flags:
```
--artifact, -a     Hardhat artifact JSON (uses deployedBytecode if present)
--bin-runtime, -r  Raw runtime bytecode (.bin) file
--function, -f     Function signature e.g. add(uint256,uint256)
--args, -A         Comma separated arguments matching the signature
--calldata, -d     Full calldata hex (overrides --function/--args)
```

Example (ABI encoding):
```bash
echoevm call -a ./artifacts/Add.json -f add(uint256,uint256) -A 2,40
```

Example (raw calldata override):
```bash
echoevm call -r ./runtime.bin -d 771602f70000000000000000000000000000000000000000000000000000000000000001
```

Output includes a structured log line with the top-of-stack value (if any).

#### 4. trace
Like `call` but emits JSON lines (one per step, or pre/post pair if `--full` is used) for inspection / tooling.

Flags:
```
--artifact, -a     Hardhat artifact path
--bin-runtime, -r  Raw runtime bytecode file
--function, -f     Function signature
--args, -A         Comma separated arguments
--calldata, -d     Full calldata hex
--limit            Stop after N steps (0 = unlimited)
--full             Emit both pre and post state for each opcode
```

Example (first 40 steps only, pre-state only):
```bash
echoevm trace -a ./artifacts/Add.json \
  -f add(uint256,uint256) -A 1,2 --limit 40 | jq .
```

Example (full pre/post):
```bash
echoevm trace -a ./artifacts/Loops.json \
  -f forLoop(uint256) -A 5 --full | jq .
```

#### 5. repl
Interactive EVM shell for experimenting with opcodes.

```bash
echoevm repl
```
Type opcodes (e.g., `PUSH1 10 ADD`) and see the stack/memory update in real-time.

#### 6. run
Execute raw bytecode with optional debug tracing.

Flags:
```
--debug         Enable step-by-step debug trace
```

Example:
```bash
echoevm run --debug 6001600201
```

#### 7. version
Display build metadata (set via `-ldflags` in the Makefile).

```
echoevm version
echoevm version --json
```

Example JSON output:
```json
{
  "version": "v0.1.0",
  "git_commit": "a1b2c3d",
  "build_date": "2025-09-14T10:10:10Z",
  "go_version": "go1.23.2",
  "platform": "darwin/arm64"
}
```

Build with custom version:
```bash
make build VERSION=v0.1.0
echoevm version
```

### ABI Encoding Support
Supported types for `--function/--args` encoding:
- uint8, uint16, ..., uint256 (decimal or 0x hex)
- int8, int16, ..., int256 (decimal or 0x hex)
- bool (true/false)
- string (UTF-8, dynamic)
- address (0x-prefixed 40 hex chars)
- bytes (0x-prefixed dynamic hex)
- bytes1, bytes2, ..., bytes32 (0x-prefixed fixed hex)
- T[] arrays (semicolon-separated values in brackets)

Array syntax example:
```bash
# uint256[] array
echoevm call -a ./artifacts/Sum.json -f sum(uint256[]) -A "[1;2;3;4;5]"

# address[] array
echoevm call -a ./artifacts/Multi.json -f send(address[]) -A "[0xabc...;0xdef...]"
```

Other Solidity types (tuples, nested arrays) are not yet enabled.

## 🔍 Examples

Basic bytecode execution:
```bash
# Simple arithmetic: PUSH1 1 PUSH1 2 ADD
echoevm run 6001600201

# With debug trace
echoevm run --debug 6001600201
```

Using ABI encoding with contract artifacts:
```bash
# Deploy a contract
echoevm deploy -a ./artifacts/Add.json --print

# Call a function
echoevm call -a ./artifacts/Add.json -f add(uint256,uint256) -A 2,40

# Generate execution trace
echoevm trace -a ./artifacts/Add.json -f add(uint256,uint256) -A 7,9 --full | head -n 20
```

## 🧪 Testing

EchoEVM includes a comprehensive testing suite:

```bash
make test             # Run unit and integration tests
make test-unit        # Run Go package unit tests
make test-integration # Run integration tests (deployment, calls, storage)
```

To run official Ethereum compliance tests (GeneralStateTests):
```bash
make setup-tests      # Download Ethereum test fixtures (~100MB)
make test-compliance  # Run compliance tests
```

## 🏗 Architecture Overview

| Layer | Path | Notes |
|-------|------|-------|
| Core Primitives | `internal/evm/core` | Stack, memory, opcode table |
| Interpreter | `internal/evm/vm` | Execution loop + trace hooks |
| CLI | `cmd/echoevm` | Cobra commands (`deploy`, `call`, `trace`) |
| Config & Constants | `internal/config` | Defaults / env variable names |
| Logging | `internal/logger` | Zerolog wrapper & helpers |

### Supported Opcode Categories
Arithmetic, Bitwise, Comparison, Stack, Memory, Storage, Control Flow, Environment, Call/Return/Revert.

## 🚦 Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Successful execution |
| 1 | Execution reverted (REVERT) |
| 2 | Invalid arguments / configuration error |

## ⚙ Configuration & Logging

Environment variables and defaults are documented in: `docs/CONFIGURATION.md` and `docs/LOGGING_GUIDE.md`.

Quick examples:
```bash
export ECHOEVM_LOG_LEVEL=debug
echoevm call -a ./artifacts/Add.json -f add(uint256,uint256) -A 3,5

export ECHOEVM_LOG_LEVEL=trace
echoevm trace -a ./artifacts/Add.json -f add(uint256,uint256) -A 1,2 --limit 10
```

`--output json` switches user-facing command output (not the trace stream) to JSON where implemented. Use `echoevm version --json` for machine-readable build info.

## 🗺 Roadmap

- Expanded ABI types (tuples, nested arrays)
- Fork-specific opcode behavior (pre/post merge, Cancun opcodes)
- Improved compliance test coverage

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`feat/<concise-topic>`)
3. Add / update tests
4. Run `make test` and ensure build is clean
5. Open a PR with a clear description + rationale

Issues / discussions for roadmap ideas are welcome.

## 📄 License

This project is open source; see `LICENSE` (or add one if missing) for terms.

---
If you are using EchoEVM in research, experiments, or education, a citation or link back is appreciated.
