# EchoEVM

[![CI](https://github.com/smallyunet/echoevm/actions/workflows/ci.yml/badge.svg)](https://github.com/smallyunet/echoevm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/smallyunet/echoevm?color=blue)](https://github.com/smallyunet/echoevm/releases)
[![Rust](https://img.shields.io/badge/rust-1.95+-dea584?logo=rust)](https://www.rust-lang.org/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

**Embedded Ethereum execution and bounded causal evidence, implemented in Rust.**

EchoEVM executes EVM bytecode, Solidity contracts, and self-contained Mainnet
transaction witnesses. It is an executor, not a wrapper around Geth, an RPC
debug method, or a remote service. Native, WebAssembly, Chrome, CLI, and editor
frontends all use the same embedded Rust engine.

[Playground](https://smallyunet.github.io/echoevm/) ·
[latest release](https://github.com/smallyunet/echoevm/releases/latest) ·
[frozen v1 protocol](protocol/v1/README.md) ·
[replay witness](docs/REPLAY_WITNESS.md)

## Install

```bash
brew install smallyunet/tap/echoevm
```

Or install the Rust CLI from a clone:

```bash
cargo install --path crates/echoevm-cli --locked
```

Tagged releases include native binaries for Linux, macOS, and Windows, a VS
Code VSIX, portable Agent Skills, and `echoevm-chrome-1.0.0.zip`.

## Execute locally

```bash
# Raw bytecode
echoevm run 60016002015f5260205ff3 --json

# Complete opcode trace or bounded evidence
echoevm trace 600160020100 --format jsonl
echoevm trace 600160020100 --format evidence-json --profile arithmetic --limit 20

# Compile, deploy, commit constructor state, and call a Solidity function
echoevm solidity run ./editors/vscode/examples/Counter.sol \
  --contract Counter --function 'increment()' --trace --format json
```

`solidity run` invokes a local `solc` through standard JSON. Contract execution
remains in the embedded EchoEVM engine.

## Replay a Mainnet transaction

```bash
echoevm replay ./transaction.witness.json \
  --format evidence-json --profile auto --limit 40
```

Replay reads only `echoevm.replay-witness.v1`. The witness contains the signed
transaction, exact block header, touched accounts, code, storage, and historical
block hashes needed by the transaction. No RPC or external executor is contacted.

For fixture acquisition only, an explicit adapter can capture a witness from a
trace-capable RPC:

```bash
echoevm witness import-debug 0x0123... \
  --rpc-url https://your-trace-rpc.example \
  --out transaction.witness.json
```

The adapter ends after writing the witness. Its upstream result is never used as
the replay result or semantic oracle.

## Browser and editor embedding

The Manifest V3 Chrome extension packages the Rust engine as WebAssembly. Load
the extracted release ZIP from `chrome://extensions`, open an Etherscan
transaction page, and select a self-contained witness. Execution happens inside
Chrome; no CLI installation is required.

The VS Code extension compiles Solidity locally and runs the same Rust CLI,
showing status, gas, source locations, storage output, and opcode evidence.

## Compatibility and conformance

The stable wire boundary is frozen under [`protocol/v1`](protocol/v1/README.md):

- `echoevm.trace.v1`
- `echoevm.evidence.v1`
- `echoevm.replay-witness.v1`
- Solidity/editor protocol version 1

The release gate pins Ethereum execution-spec fixtures at `tests@v20.0.1` and
executes the authored Prague and Osaka suites under Osaka rules. The required
summary is exactly 187 files, 3,461 transactions, 3,244 accepted, 217 rejected,
and zero skipped. Official fixtures are the oracle; the archived Go
implementation is not.

Supported transaction/interpreter scope is Cancun through Osaka. Pre-Cancun
replay, full block validation, consensus networking, and Prague request
processing are outside the v1 claim. Evidence is diagnostic output, not a
security audit or formal proof.

## Workspace

| Crate | Purpose |
|---|---|
| `echoevm-protocol` | Stable JSON/witness types and limits |
| `echoevm-core` | Embedded execution, tracing, state and transaction replay |
| `echoevm` | Native CLI, Solidity, witness acquisition and local Web UI |
| `echoevm-wasm` | Browser-safe Wasm bindings |

The pre-rewrite Go tree is permanently retained on the [`go`](https://github.com/smallyunet/echoevm/tree/go)
branch. `main` contains only the Rust implementation.

## Development

```bash
make build
make test
make test-conformance-full
```

`make test-conformance-full` downloads and verifies the pinned official fixture
archive before executing the zero-skip gate.

EchoEVM is available under the [MIT License](LICENSE).
