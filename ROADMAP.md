# EchoEVM roadmap

**Current release: v1.0.0 — Rust engine and protocol v1**

## Delivered in v1.0.0

- Frozen trace, evidence, witness, Solidity, command, and status contracts.
- Permanent pre-rewrite snapshot on the `go` branch.
- Rust native executor for Cancun, Prague, and Osaka semantics.
- Signed EIP-2718 transaction decoding, sender recovery, block context, prestate,
  state commit, nested execution, precompiles, and self-contained replay.
- Rust CLI for raw bytecode, deploy/call, trace, evidence, Solidity, witness,
  REPL, and local Web execution.
- Browser-safe Rust Wasm and a Manifest V3 Chrome extension without a local CLI.
- VS Code client protocol, native release binaries, Docker/GHCR, Homebrew, and
  GitHub release packaging.
- Pinned `tests@v20.0.1` official gate: 187 files, 3,461 transactions, zero skip.

## Next

- Expand the frozen official corpus to later execution-spec releases without
  shrinking the current gate.
- Add complete historical `BLOCKHASH` acquisition and proof-backed witness
  construction independent of debug namespaces.
- Add Prague request/system-call and full block transition conformance.
- Add source-level stepping and richer storage/value-flow links while preserving
  the bounded evidence schema.
- Publish the browser extension through a signed store channel after store
  credentials and review are available.

Passing a fixture corpus is a scoped compatibility claim, not proof of full
Ethereum client equivalence.
