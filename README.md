# EchoEVM

[![CI](https://github.com/smallyunet/echoevm/actions/workflows/ci.yml/badge.svg)](https://github.com/smallyunet/echoevm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/smallyunet/echoevm?color=blue)](https://github.com/smallyunet/echoevm/releases)
[![Rust](https://img.shields.io/badge/rust-1.95+-dea584?logo=rust)](https://www.rust-lang.org/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Playground](https://img.shields.io/badge/playground-GitHub_Pages-34d399)](https://smallyunet.github.io/echoevm/)

**Independent Ethereum execution with exact traces and bounded evidence, implemented in Rust.**

EchoEVM executes EVM bytecode, Solidity contracts, and self-contained Mainnet
transaction witnesses. It is an executor, not a wrapper around Geth, an RPC
debug method, a remote service, or another EVM implementation. Native,
WebAssembly, Chrome, CLI, and editor frontends all use EchoEVM's own Rust
interpreter, state transition, call-frame, gas, fork, and precompile code.

[Playground](https://smallyunet.github.io/echoevm/) ·
[latest release](https://github.com/smallyunet/echoevm/releases/latest) ·
[documentation](docs/README.md) ·
[architecture](docs/ARCHITECTURE.md) ·
[frozen v1 protocol](protocol/v1/README.md)

Use EchoEVM when you need to:

- execute raw EVM bytecode or a Solidity call without starting a JSON-RPC node;
- inspect an exact opcode trace, or select a bounded diagnostic view from that
  already-complete trace;
- replay a transaction from an explicit, self-contained historical witness; or
- embed the same Rust execution kernel in native, Wasm, Chrome, or VS Code tools.

EchoEVM is not a full Ethereum node, RPC fork, static analyzer, formal verifier,
or drop-in replacement for a production execution client.

## Quick start

```bash
brew install smallyunet/tap/echoevm
echoevm run 60016002015f5260205ff3 --json
```

Or install the Rust CLI from a clone:

```bash
cargo install --path crates/echoevm-cli --locked
```

Release packaging includes native binaries for Linux, macOS, and Windows, a VS
Code VSIX, the portable `echoevm-debug` Agent Skill, and the Chrome extension
ZIP. The contributor-only `echoevm-conformance` skill remains in the
repository.

## Core workflows

```bash
# Complete opcode trace or bounded evidence
echoevm trace 600160020100 --format jsonl
echoevm trace 600160020100 --format evidence-json --profile arithmetic --limit 20

# Compile, deploy, commit constructor state, and call a Solidity function
echoevm solidity run ./editors/vscode/examples/Counter.sol \
  --contract Counter --function 'increment()' --trace --format json
```

`solidity run` invokes a local `solc` through standard JSON. Contract execution
remains in the embedded EchoEVM engine.

## Explain an execution

```bash
# Explain a self-contained transaction witness
echoevm explain replay ./transaction.witness.json --format text

# Explain a self-contained call-level test
echoevm explain test ./failure.test-witness.json --format text

# Deploy a linked Foundry artifact, run setUp(), and explain one test call
echoevm explain foundry out/Counter.t.sol/CounterTest.json \
  --test 'testIncrement()' --witness-out failure.test-witness.json \
  --format text

# Explain one compiled Solidity function and compare its ABI return value
echoevm explain solidity ./Contract.sol \
  --contract Contract --function 'average(uint256[])' --args '[2,4,6,8]' \
  --expect-return 0x05 --format json
```

`echoevm explain` emits either a human-readable report or the stable
`echoevm.explanation.v1` document. It separates the verdict, directly captured
causal findings, an optional root cause, and limitations. When an observed
result differs from a declared expectation but the selected evidence cannot
establish why, the verdict is `insufficient-evidence`.

Call-level tests use the strict `echoevm.test-witness.v1` protocol. It carries
runtime bytecode, calldata, explicit accounts/storage/caller/value/environment,
expectations, and optional source locations. `explain foundry` executes linked
constructor bytecode and an ABI-visible zero-argument `setUp()` locally, closes
the final call's read set against the resulting isolated state, then replays the
materialized witness independently. Standard or dynamically reached HEVM
cheatcodes fail with `unsupported-capability`; RPC forks and external historical
state are not inferred.

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

For the first transaction in a block, EchoEVM can instead use standard RPC
methods only. It uses `eth_createAccessList` as an optional accelerator, then
replays locally to discover missing reads to a bounded fixed point. Every
account and storage value is fetched with EIP-1186 proofs from the parent block,
verified against the parent state root, and checked against fetched code before
the same self-contained replay contract is written:

```bash
echoevm witness import-proof 0x0123... \
  --rpc-url https://your-rpc.example \
  --out transaction.witness.json \
  --proofs-out transaction.proofs.json
```

Standard RPC exposes block-boundary proofs, not intermediate state between
transactions. `import-proof` therefore fails closed unless `transactionIndex`
is zero. It never silently substitutes post-block state.

## Browser and editor embedding

The Manifest V3 Chrome extension packages the Rust engine as WebAssembly. On a
verified Etherscan contract page, Contract Lens reads the displayed ABI and
deployed bytecode and locally executes ABI functions marked `pure` in an
explicit empty-state sandbox. On a transaction page, users can select a
self-contained witness for exact standalone replay. Execution happens inside
Chrome; no CLI installation is required.

The VS Code extension compiles Solidity locally and runs the same Rust CLI,
showing status, gas, source locations, storage output, and opcode evidence.

The complete trace is the execution record. Bounded evidence is a deterministic
post-execution selection for a chosen profile; it does not alter execution and
emits only frame, rollback, or tracked value-flow links established by captured
execution facts.

## Compatibility and conformance

The stable wire boundary is frozen under [`protocol/v1`](protocol/v1/README.md):

- `echoevm.trace.v1`
- `echoevm.evidence.v1`
- `echoevm.replay-witness.v1`
- Solidity/editor protocol version 1

The current `main` gate pins Ethereum execution-spec fixtures at
`tests@v20.0.1` and executes the complete matching state-test directories with
zero skip: 2,337/11,554 Cancun, 2,471/13,851 Prague, and 2,408/14,516 Osaka
files/transactions. Across 39,921 transactions it checks canonical signed
transaction bytes and sender recovery, exact accept/reject category, receipt
status and gas, logs hash, post-state accounts, and post-state root.
A shared native/Wasm bytecode matrix adds 15 exact vectors across 11 semantic
categories and freezes EchoEVM's 170-name opcode inventory. See the
[`conformance contract`](docs/CONFORMANCE.md) and
[`bytecode compatibility contract`](docs/BYTECODE_COMPATIBILITY.md). Official
fixtures are the oracle; the archived Go implementation is not. “A-grade” is
an EchoEVM release-gate label, not an Ethereum Foundation certification.

Supported transaction/interpreter scope is Cancun through Osaka. Pre-Cancun
replay, full block validation, consensus networking, and Prague request
processing are outside the v1 claim. Evidence is diagnostic output, not a
security audit or formal proof.

## Documentation

| Goal | Start here |
|---|---|
| Find the right guide | [Documentation index](docs/README.md) |
| Replay a complete historical transaction | [Replay witnesses](docs/REPLAY_WITNESS.md) |
| Integrate JSON or JSONL output | [Trace protocol](docs/TRACE_PROTOCOL.md) |
| Review semantic coverage | [Bytecode compatibility](docs/BYTECODE_COMPATIBILITY.md) |
| Audit the release conformance claim | [Conformance contract](docs/CONFORMANCE.md) |
| Understand the internal module boundaries | [Architecture](docs/ARCHITECTURE.md) |
| Build against stable schemas | [Protocol v1](protocol/v1/README.md) |

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
make test-bytecode-conformance
make test-conformance-full
```

`make test-conformance-full` downloads and verifies the pinned official fixture
archive before executing the zero-skip gate.

## Echo family

| Project | Execution domain | Static playground |
|---|---|---|
| [EchoEVM](https://github.com/smallyunet/echoevm) | Solidity and EVM bytecode | [Open](https://smallyunet.github.io/echoevm/) |
| [EchoSVM](https://github.com/smallyunet/echosvm) | Solana transactions and sBPF | [Open](https://smallyunet.github.io/echosvm/) |
| [EchoRV](https://github.com/smallyunet/echorv) | RISC-V firmware and traces | [Open](https://smallyunet.github.io/echorv/) |
| [EchoScript](https://github.com/smallyunet/echoscript) | Bitcoin Tapscript inputs | [Open](https://smallyunet.github.io/echoscript/) |

Each project executes locally, emits a versioned evidence schema, and publishes
frozen reproducible cases through the same static playground contract.

EchoEVM is available under the [MIT License](LICENSE).
