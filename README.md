# EchoEVM

[![CI](https://github.com/smallyunet/echoevm/actions/workflows/ci.yml/badge.svg)](https://github.com/smallyunet/echoevm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/smallyunet/echoevm?style=flat&color=blue)](https://github.com/smallyunet/echoevm/releases)
[![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat)](LICENSE)
[![Live Demo](https://img.shields.io/badge/live_demo-r.dark20.xyz-orange)](https://r.dark20.xyz/)

EchoEVM executes Solidity and EVM bytecode and emits bounded causal evidence for
AI coding agents. It explains nested calls, reverted writes, value flow, storage
changes, gas usage, and failure causes without flooding the model with a full
opcode trace.

Use it to diagnose one execution locally, compare behavior with embedded Geth,
or replay a confirmed Ethereum Mainnet transaction from RPC prestate.

Try the hosted [Transaction Explainer](https://r.dark20.xyz/).

## Why EchoEVM

- **Agent-sized evidence** — `echoevm.evidence.v1` selects the state, call,
  failure, ABI, gas, or arithmetic events relevant to the question.
- **Causal execution links** — `enters-frame`, `returns-to`, `rolls-back`, and
  `value-flow` connect effects across nested frames and stack transformations.
- **Full execution, bounded output** — filters and limits reduce presentation;
  they do not stop execution early or change its result.
- **Optional Geth comparison** — differential execution is a conformance tool,
  not a prerequisite for the primary explain-one-input workflow.
- **Real transaction replay** — hydrate transaction prestate through a
  trace-capable Ethereum Mainnet RPC and inspect the complete call tree.

## Quick Start

Install the CLI on macOS:

```bash
brew install smallyunet/tap/echoevm
```

Or install from source with Go 1.25+:

```bash
go install github.com/smallyunet/echoevm/cmd/echoevm@latest
```

Solidity source execution also requires a compatible `solc` executable on
`PATH`. From a cloned repository, run the included counter example:

```bash
echoevm solidity run ./editors/vscode/examples/Counter.sol \
  --contract Counter \
  --function 'increment()' \
  --format evidence-json \
  --profile storage \
  --limit 40
```

The result is a stable `echoevm.evidence.v1` document containing execution
metadata, selected opcode effects, causal links, and explicit truncation
metadata. Use `jq` to inspect only the evidence selected for the run:

```bash
echoevm solidity run ./editors/vscode/examples/Counter.sol \
  --contract Counter \
  --function 'increment()' \
  --format evidence-json \
  --profile storage \
  --limit 40 | jq '{schema, execution, events, links, selection}'
```

## Core Workflows

### Diagnose Solidity source

Compile, deploy, and call one contract function in an isolated Osaka state:

```bash
echoevm solidity run ./Contract.sol \
  --contract Contract \
  --constructor-args 7 \
  --function 'read()' \
  --format evidence-json \
  --profile auto \
  --limit 40
```

Available evidence profiles are `auto`, `revert`, `storage`, `call`, `abi`,
`gas`, `arithmetic`, and `full`.

EchoEVM automatically reads `foundry.toml` and `remappings.txt` for remappings,
optimizer settings, optimizer runs, and `via_ir`. Explicit flags are available
for non-Foundry workspaces. Deployment and runtime gas limits can be controlled
independently with `--deploy-gas` and `--gas`.

### Inspect the full opcode process

Use `trace` when compact causal evidence is not enough:

```bash
echoevm trace \
  --bin-runtime ./runtime.bin \
  --calldata 0x1234 \
  --around-step 42 \
  --window 5 \
  --format json
```

Trace output follows `echoevm.trace.v1` and can include stack deltas, bounded
memory changes, storage context, gas breakdown, control flow, halt state, and
deterministic explanations. Filter by opcode, depth, step range, or event field.

For a compact view of raw runtime bytecode:

```bash
echoevm trace \
  --bin-runtime ./runtime.bin \
  --calldata 0x1234 \
  --profile storage \
  --limit 40 \
  --format evidence-json
```

See [Trace Protocol](docs/TRACE_PROTOCOL.md) for schema semantics, selection
behavior, and the recommended agent workflow.

### Compare with embedded Geth

```bash
echoevm diff \
  --code 60026003015f5260205ff3 \
  --input 0x \
  --gas 1000000 \
  --fork Osaka

echoevm diff --code 00 --format summary-json
```

Both engines run under the selected ruleset from Frontier through Osaka; Osaka
is the default. A `MATCH` applies only to the tested input and environment; it
is not a claim of complete EVM compatibility.

Start the local Transaction Explainer with:

```bash
echoevm diff --web --addr :8080
```

### Replay a Mainnet transaction

Replay requires an RPC endpoint with `debug_traceTransaction` and the built-in
`prestateTracer` enabled:

```bash
echoevm replay 0x0123... \
  --rpc-url https://your-trace-rpc.example \
  --format evidence-json \
  --profile auto \
  --limit 40

ECHOEVM_ETHEREUM_RPC=https://your-trace-rpc.example \
  echoevm replay https://etherscan.io/tx/0x0123... \
    --format evidence-json \
    --profile revert
```

Replay evidence uses the same profiles, causal links, and presentation limits
as local source execution. It also carries transaction/fork provenance,
EchoEVM/Geth comparison confidence, and compatibility warnings without
duplicating both engines' complete traces. Use `--format json` when the full
differential replay is required.

EchoEVM recognizes confirmed Ethereum Mainnet transactions and selects Cancun,
Prague, or Osaka transaction/interpreter rules from the block timestamp.
Pre-Cancun transactions retain an explicit compatibility warning.

### Run raw bytecode

```bash
# PUSH1 1, PUSH1 2, ADD
echoevm run 6001600201

# Print a step-by-step debug trace
echoevm run --debug 6001600201
```

## Output Formats

| Format | Use it for |
|---|---|
| `evidence-json` | Bounded causal evidence for diagnosis and agent context |
| `summary-json` | Compact execution or differential verdicts without opcode arrays |
| `json` | Complete structured command output |
| `jsonl` | Streaming full opcode events from `trace` |
| `text` | Human-readable terminal output |

`--limit` bounds emitted evidence or trace events while execution still runs to
completion. Output metadata reports total, selected, omitted, and truncated
counts so consumers can distinguish complete execution from partial display.

## Commands

| Command | Description |
|---|---|
| `solidity inspect` | List deployable contracts and ABI functions as versioned JSON |
| `solidity run` | Compile, deploy, and call one Solidity function |
| `trace` | Emit explainable, filterable opcode or causal evidence |
| `diff` | Compare EchoEVM with embedded Geth |
| `replay` | Replay a confirmed transaction from RPC prestate |
| `run` | Execute raw bytecode or a transaction fixture |
| `deploy` | Execute constructor bytecode and extract runtime code |
| `call` | Execute runtime bytecode with ABI encoding |
| `disasm` | Disassemble bytecode |
| `repl` | Start the interactive EVM shell |
| `web` | Start the browser-based visual debugger |
| `version` | Display version and build metadata |

Run `echoevm <command> --help` for the authoritative flags and examples.

## Agent and Editor Integrations

The repository includes two read-only Agent Skills:

- [`echoevm-debug`](.agents/skills/echoevm-debug/SKILL.md) compiles and executes
  Solidity, compares bytecode with Geth, and replays confirmed Mainnet
  transactions.
- [`echoevm-conformance`](.agents/skills/echoevm-conformance/SKILL.md) validates
  interpreter changes with focused tests, pinned official fixtures, and
  differential vectors.

Codex and Gemini CLI discover the canonical Skills under `.agents/skills`.
Claude Code uses the synchronized copies under `.claude/skills`. Tagged GitHub
releases also include installable `.skill` archives.

The [VS Code extension](editors/vscode/README.md) adds Run and Compare CodeLens
actions above Solidity functions, shows the latest status and gas result beside
the source, reports concrete execution failures in Problems, and organizes
state, comparison, and key-opcode evidence in a source-navigable side view.
The complete opcode table remains available on demand, without starting a
JSON-RPC node.

Local Solidity source is not sent to the hosted Explorer by these integrations.
The hosted Transaction Explainer accepts public transaction hashes, exposes
question-routed evidence first, and keeps the complete differential trace in an
advanced disclosure. Successful explanations have shareable `/tx/<hash>` URLs.

## Scope and Limitations

- Transaction and interpreter semantics are declared from Cancun through Osaka;
  Prague system requests, full block validation, consensus networking, and
  historical `BLOCKHASH` witnesses are not implemented.
- A matching differential result proves only the tested input and environment.
- Evidence is execution diagnostics, not a security audit or formal
  verification result.
- Solidity execution does not implement Foundry cheatcodes, RPC forking,
  payable calls, source-level stepping, or test discovery. Source-run output
  does include compiler source ranges and runtime PC mappings for editor
  evidence.
- Mainnet replay requires a trace-capable RPC; EchoEVM does not approximate
  transaction prestate from the parent block.
- Trie-backed state supports lazy reads; committing modified state roots is not
  yet supported.

## Evidence and Benchmarks

The published v0.0.41 compiled-Solidity benchmark covers nested REVERT, CREATE,
DELEGATECALL, and arithmetic failures across 36 scored agent runs. On its frozen
cases and model configuration, routed evidence produced 11/12 strict diagnoses
versus 8/12 for broad opcode context while using 39.8% fewer fresh tokens.

These results describe the frozen benchmark, not general diagnostic accuracy or
complete EVM compatibility. See the
[benchmark methodology and artifacts](benchmarks/trace-value-v2/README.md).
Mainnet replay evidence has deterministic regression coverage, but no new
external-model accuracy or token-savings result is claimed for real
transactions yet. Its frozen-witness acceptance gate is documented in the
[Mainnet replay evidence benchmark](benchmarks/replay-evidence/README.md).

## Development

```bash
git clone https://github.com/smallyunet/echoevm.git
cd echoevm
make build
make test
```

Focused validation commands:

```bash
make test-unit
make test-integration
make test-compliance
make test-differential
make test-conformance
make test-conformance-full
make test-skills
```

See [ROADMAP.md](ROADMAP.md) for delivered versions and current priorities.
Issues, discussions, and pull requests are welcome.

## License

EchoEVM is available under the [MIT License](LICENSE).
