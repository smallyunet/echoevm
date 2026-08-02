# echoevm

[![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/smallyunet/echoevm?style=flat&color=blue)](https://github.com/smallyunet/echoevm/releases)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen?style=flat)]()
[![Live Demo](https://img.shields.io/badge/live_demo-r.dark20.xyz-orange)](https://r.dark20.xyz/)

**EchoEVM** is a minimal, pedagogical Ethereum Virtual Machine (EVM) implementation written in Go. It focuses on transparent bytecode execution, traceability, and ease of experimentation rather than production consensus or networking features.

Try the hosted Differential Explorer at **[r.dark20.xyz](https://r.dark20.xyz/)**.

---

## 📑 Table of Contents

- [What's New in v0.0.34](#-whats-new-in-v0034)
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

## 🆕 What's New in v0.0.34

- **Verified CLI Downloads**: Tagged releases build platform-specific CLI assets and publish a SHA-256 manifest for editor-managed installation.
- **Zero-Terminal VS Code Onboarding**: Toolchain health, verified CLI installation, bundled `solc-js 0.8.30`, and a ready-to-run Solidity example reduce setup friction.
- **Portable Solidity Compilation**: The runner uses Standard JSON so native `solc`, `solcjs`, and the bundled compiler share one versioned execution protocol.

### Previous v0.0.33

- **Correct Marketplace Identity**: The VS Code extension now publishes as `smallyu.echoevm` under the project's established `smallyu` publisher.

### Previous v0.0.32

- **Solidity Source Runner**: Compile a local Solidity source, deploy constructor state, call one ABI function, and optionally compare execution and traces with embedded Geth.
- **Versioned Editor Protocol**: Inspect contracts and ABI functions through compact schema-v1 JSON with structured errors and cancellable execution.
- **VS Code Extension MVP**: Select, run, and compare Solidity functions directly from `.sol` editors, with configurable compiler paths, execution output, and an on-demand opcode trace panel.

### Previous v0.0.31

- **Real STATICCALL Protection**: Read-only mode now propagates through nested frames and rejects storage, transient-storage, log, contract-creation, value-transfer, and self-destruct mutations.
- **Correct Contract-Creation Rollback**: Failed `CREATE` and `CREATE2` executions restore accounts, balances, persistent storage, and transient storage while preserving the creator nonce required by Ethereum semantics.
- **Geth-Matched Creation Gas**: Initcode word charges, EIP-150 forwarding, REVERT refunds, exceptional-halt burns, runtime code-deposit cost, size limits, and invalid code prefixes now match Cancun Geth behavior.
- **Nested Differential Matrix**: CALL, STATICCALL, CREATE, and CREATE2 success, REVERT, exceptional halt, gas, and state outcomes are checked against embedded Geth v1.17.4.

### Previous v0.0.30

- **Comparable Gas Diagnostics**: Transaction traces compare per-opcode gas where EchoEVM and Geth expose the same semantics, while nested calls and creates are labeled as not comparable instead of producing false divergences.

### Previous v0.0.29

- **Verifiable Release Identity**: Production image builds fetch release tags so `echoevm version` reports the published semantic version alongside its full commit hash.

### Previous v0.0.28

- **Reliable Edge Readiness**: Deployment now checks Caddy through its local administration endpoint, avoiding false failures caused by probing an HTTPS origin by IP without matching TLS SNI.

### Previous v0.0.27

- **Correct SSTORE Gas**: Warm storage writes no longer receive an extra 100-gas charge on top of the EIP-2200 baseline.
- **Reliable Opcode Comparison**: Geth's `KECCAK256` trace name is normalized to EVM `SHA3`, and transaction traces now compare per-opcode gas cost explicitly.
- **Recent Mainnet Transactions**: Opening the Explorer loads five transactions from the latest Ethereum block through a short server-side cache; selecting one only fills the replay input.
- **Light Differential Explorer**: The complete replay, warning, divergence, and trace interface now uses an accessible light color system.

### Previous v0.0.26

- **Replay Readiness**: `/readyz` verifies Ethereum Mainnet, `prestateTracer`, and opcode-trace support before a deployment is accepted.
- **Atomic Deployment Bundle**: Immutable images carry Compose, Caddy, and deployment configuration; the server validates and atomically activates the bundle with whole-stack rollback.
- **Credential Preservation**: Deployments retain all existing environment settings, keep `.env` at mode `0600`, and never replace RPC credentials with image-only configuration.
- **Actionable HTTP Errors**: Replay distinguishes invalid input, missing or pending transactions, upstream failures, unavailable trace capabilities, and timeouts.

### Previous v0.0.25

- **Mainnet-Only Replay**: Transaction hashes and Etherscan URLs now resolve consistently to Ethereum Mainnet, and non-mainnet RPC endpoints are rejected before transaction lookup.

### Previous v0.0.24

- **Reliable Explorer Assets**: Versioned JavaScript and CSS URLs prevent CDN caches from pairing a new transaction-replay page with an older script that lacks the replay button handler.

### Previous v0.0.23

- **Lint-Clean Release**: Normalized replay parser errors for the repository's staticcheck contract; behavior is unchanged from the transaction replay implementation introduced in v0.0.22.

- **Transaction Replay**: Paste a transaction hash or Etherscan URL in the Explorer, hydrate exact execution prestate through `prestateTracer`, and compare status, output, gas, post-state, and instructions.
- **Full Call-Frame Tracing**: Opcode hooks now propagate through nested `CALL`, `DELEGATECALL`, `STATICCALL`, `CREATE`, and `CREATE2` frames.
- **Replay CLI**: `echoevm replay` exposes the same transaction-level engine with text or JSON output.
- **Safer RPC Integration**: Explorer links are parsed through an allowlist, RPC credentials remain server-side, and unsupported forks are reported explicitly.

### Previous v0.0.22

- Introduced RPC-backed transaction replay, Etherscan input, nested call-frame tracing, and post-state comparison.

### Previous v0.0.21

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
| **Replay** | Transaction hash/Etherscan input, RPC prestate hydration, nested call-frame comparison |
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

- Go 1.25+
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

### Compile and run a Solidity function

The Solidity runner invokes a local `solc`, deploys the selected contract in an
isolated Cancun state, and then calls one ABI function. Constructor state and
immutable values are preserved for the call.

```bash
echoevm solidity run ./Calculator.sol \
  --contract Calculator \
  --constructor-args 7 \
  --function 'add(uint256,uint256)' \
  --args 2,40

# Compare the deployed call with embedded Geth and include opcode traces
echoevm solidity run ./Calculator.sol \
  --contract Calculator \
  --constructor-args 7 \
  --function 'read()' \
  --diff --trace

# Stable machine-readable output for editor and npm integrations
echoevm solidity run ./Calculator.sol --constructor-args 7 -f 'read()' --format json
```

The base path defaults to the source directory. Use `--base-path` and repeated
`--include-path` flags for imports. Compilation is pinned to the Cancun EVM
target; the MVP does not implement Foundry cheatcodes, RPC forking, payable
calls, source maps, or test discovery.

### VS Code extension MVP

The workspace extension in `editors/vscode` uses the versioned Solidity JSON
protocol to inspect contracts, select an ABI function, run or compare it, and
show an on-demand opcode trace without starting a JSON-RPC node.

```bash
cd editors/vscode
npm ci
npm run package:vsix
code --install-extension echoevm-0.0.1.vsix
```

The extension expects `echoevm` and `solc` on `PATH`. Override them with
`echoevm.executablePath` and `echoevm.solcPath`. In Remote SSH, Dev Containers,
or WSL, those executables must be installed in the remote workspace runtime.

### Compare EchoEVM with embedded Geth

```bash
echoevm diff \
  --code 60026003015f5260205ff3 \
  --input 0x \
  --gas 1000000

# Machine-readable output
echoevm diff --code 00 --format json

# Local Differential Explorer
echoevm diff --web --addr :8080
```

The differential engine runs both implementations under Cancun rules with
isolated in-memory state. A `MATCH` applies only to that input and environment;
it is not a claim that EchoEVM is completely EVM-compatible.

### Replay a real transaction

Transaction replay requires an RPC endpoint with `debug_traceTransaction` and
the built-in `prestateTracer` enabled.

```bash
echoevm replay 0x0123... --rpc-url https://your-trace-rpc.example
echoevm replay https://etherscan.io/tx/0x0123... --format json

ECHOEVM_ETHEREUM_RPC=https://your-trace-rpc.example \
  echoevm diff --web --addr :8080
```

The Explorer keeps raw bytecode comparison under its Advanced section. Replay
supports confirmed Ethereum Mainnet transactions. EchoEVM currently
executes Cancun rules; transactions from other forks remain inspectable but are
marked with a compatibility warning.

### Server deployment

Production deployment uses Docker Compose and
`.github/workflows/deploy-server.yml`. Deployments are manual: starting the
workflow runs the test suite, publishes an immutable Linux/amd64 image to GHCR,
then asks a restricted server wrapper to pull and activate that digest. The
image carries the matching Compose, Caddy, and deployment files; the wrapper
validates them, preserves the complete environment file, starts the whole
stack, and requires `/readyz` to confirm Mainnet trace capability. Pushes
to `main` and release tags do not deploy automatically. Compose applies a
non-root user, read-only filesystem, dropped capabilities, resource limits, and
a `/healthz` liveness check. A failed stack or readiness check restores the
previous image, configuration, environment, and deployment wrapper. Its
dedicated root SSH key is bound to a
forced command and cannot open a shell, forward ports, or run arbitrary
commands; the existing operator SSH key is never copied to GitHub.
Set `ECHOEVM_ETHEREUM_RPC` in the deployment environment to enable replay; the
endpoint must expose `debug_traceTransaction` and `prestateTracer`.

Upgrading a server from v0.0.25 or earlier requires one bootstrap update of the
host wrapper before running the first v0.0.26 deployment:

```bash
sudo install -m 0755 deploy/deploy-image.sh /usr/local/sbin/deploy-echoevm
sudo chmod 0600 /opt/echoevm/.env
```

After that bootstrap, each immutable application image updates the deployment
bundle and wrapper for subsequent releases. Keep `ECHOEVM_ETHEREUM_RPC` in
`/opt/echoevm/.env`; deployment changes only `ECHOEVM_IMAGE`.

The production Compose stack includes Caddy as an HTTPS origin on port 8080
for `r.dark20.xyz`. Proxy that hostname through Cloudflare, use SSL/TLS mode
`Full`, and configure an Origin Rule that rewrites the destination port to
8080. EchoEVM itself remains available only inside the Docker network.

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
| `solidity inspect` | Compile Solidity and return versioned contract/function metadata |
| `solidity run` | Compile, deploy, and call a Solidity contract with optional trace/diff |
| `run` | Execute raw bytecode with optional debug tracing |
| `diff` | Compare results and normalized traces with embedded Geth |
| `replay` | Replay a confirmed transaction from RPC prestate |
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
--rpc-url         Ethereum RPC endpoint; replay requires debug tracing
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

The v0.0.26 baseline contains 9 pinned official Cancun cases and 17 geth
differential vectors across arithmetic, bitwise, control, crypto, environment,
fault, memory, and storage. Both suites fail on missing metadata, shrinking
case counts, missing required categories, or skipped execution.

---

## 🏗 Architecture

```
echoevm/
├── cmd/echoevm/     # CLI commands (deploy, call, trace, etc.)
├── editors/vscode/  # VS Code workspace extension and local VSIX packaging
├── internal/
│   ├── differential/  # Reusable EchoEVM/Geth runners and comparison engine
│   ├── replay/        # Transaction input parser, RPC prestate, and replay engine
│   ├── evm/
│   │   ├── core/    # Stack, memory, opcode table
│   │   └── vm/      # Interpreter, opcode implementations
│   ├── config/      # Constants, environment variables
│   ├── logger/      # Zerolog wrapper
│   └── errors/      # Error definitions
└── tests/           # Integration and compliance tests
```

### Supported Opcode Categories

Arithmetic, Bitwise, Comparison, Stack, Memory, Storage, Control Flow, Environment, Call/Return/Revert, Logging, System.

---

## ⚙️ Configuration

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
