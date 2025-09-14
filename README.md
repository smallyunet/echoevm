git clone https://github.com/smallyunet/echoevm.git
# echoevm

EchoEVM is a minimal, pedagogical Ethereum Virtual Machine (EVM) implementation written in Go. It focuses on transparent bytecode execution, traceability, and ease of experimentation rather than production consensus or networking features.

> Status: Active development. The Cobra-based CLI currently ships with `deploy`, `call`, and `trace`. Legacy documentation for `run`, `block`, `range`, and `serve` refers to planned / experimental features and is being consolidated.

## ✨ Features

- **Constructor Deployment**: Execute constructor bytecode and extract emitted runtime code (`deploy`).
- **Runtime Calls**: Execute deployed / runtime bytecode with ABI encoded calldata (`call`).
- **ABI Convenience**: Lightweight ABI function selector & argument encoding for common primitive types.
- **Execution Tracing**: JSON structured per-opcode tracing with optional pre/post state (`trace`).
- **Deterministic Core**: Small, auditable interpreter with clear stack & memory semantics.
- **Testing Suite**: Unified script plus Go unit tests covering opcodes, stack, memory, control and ABI paths.
- **Structured Logging**: Zerolog based, selectable output format (plain | json) and adjustable log level.

Planned / in-progress (roadmap): disassembler (`disasm`), JSON-RPC serving (`serve`), historical block replay (`block` / `range`), version & build info (`version`).

## ✅ Requirements

- Go 1.23.2+
- (Optional) Node.js + Hardhat (only if you want to rebuild / extend the sample Solidity contracts in `test/contract`)
- (Optional) `solc` if compiling standalone `.sol` files directly.

## 🔧 Installation

Clone and build from source:

```bash
git clone https://github.com/smallyunet/echoevm.git
cd echoevm
make build          # builds ./bin/echoevm

# (optional) install into GOPATH/bin
make install
```

Quick build without Makefile helpers:

```bash
go build -o bin/echoevm ./cmd/echoevm
```

Verify:

```bash
./bin/echoevm --help
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
echoevm deploy -a ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json --print
```

#### 2. call
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
echoevm call -a ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json \
  -f add(uint256,uint256) -A 2,40
```

Example (raw calldata override):
```bash
echoevm call -r ./runtime.bin -d 771602f70000000000000000000000000000000000000000000000000000000000000001
```

Output includes a structured log line with the top-of-stack value (if any).

#### 3. trace
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
echoevm trace -a ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json \
  -f add(uint256,uint256) -A 1,2 --limit 40 | jq .
```

Example (full pre/post):
```bash
echoevm trace -a ./test/contract/artifacts/contracts/03-control-flow/Loops.sol/Loops.json \
  -f forLoop(uint256) -A 5 --full | jq .
```

#### 4. version
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
./bin/echoevm version
```

### ABI Encoding Support
Currently supported primitive types for `--function/--args` encoding:
- uint256 / int256 (decimal or 0x hex)
- bool (true/false)
- string (UTF-8, dynamic)

Other Solidity types (arrays, bytesN, address, etc.) are not yet enabled in the helper and will return an error if used.

## 🔍 Examples

Factorial (recursive / iterative sample):
```bash
echoevm call -a ./test/contract/artifacts/contracts/01-data-types/Fact.sol/Fact.json \
  -f fact(uint256) -A 5
```

Trigger a revert (expect exit code 1):
```bash
echoevm call -a ./test/contract/artifacts/contracts/03-control-flow/Require.sol/Require.json \
  -f test(uint256) -A 0 || echo "(reverted as expected)"
```

Generate runtime from constructor and then call it:
```bash
echoevm deploy -a ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json --out-file add.runtime
echoevm call -r add.runtime -d 771602f70000000000000000000000000000000000000000000000000000000000000001
```

Trace with pre/post states:
```bash
echoevm trace -a ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json \
  -f add(uint256,uint256) -A 7,9 --full | head -n 20
```

## 🧪 Testing

Integration & unit tests are included. See `docs/TESTING_QUICK.md` or below for a summary:

```bash
./test/test.sh            # all integration tests
./test/test.sh --binary   # only raw .bin tests
./test/test.sh --contract # only Hardhat artifact tests
make test-unit            # Go package tests
```

Coverage report:
```bash
make coverage
```

More details: [test/README.md](test/README.md)

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
echoevm call -a ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -f add(uint256,uint256) -A 3,5

export ECHOEVM_LOG_LEVEL=trace
echoevm trace -a ./test/contract/artifacts/contracts/01-data-types/Add.sol/Add.json -f add(uint256,uint256) -A 1,2 --limit 10
```

`--output json` switches user-facing command output (not the trace stream) to JSON where implemented. Use `echoevm version --json` for machine-readable build info.

## 🗺 Roadmap (Short Term)

- disasm: human-readable bytecode disassembly
- serve: lightweight JSON-RPC (eth_*) sandbox
- block / range: replay selective mainnet blocks for educational analysis
- (done) version: embed commit / build info
- Expanded ABI types (address, bytes, arrays)
- Gas accounting & metering (currently simplified / placeholder in several paths)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`feat/<concise-topic>`)
3. Add / update tests (Go + integration script)
4. Run `make test-all` and ensure lint & build are clean
5. Open a PR with a clear description + rationale

Issues / discussions for roadmap ideas are welcome.

## 📄 License

This project is open source; see `LICENSE` (or add one if missing) for terms.

---
If you are using EchoEVM in research, experiments, or education, a citation or link back is appreciated.
