# EchoEVM roadmap

**Current release: v1.4.0 — proof-verified witness acquisition**

## Delivered in v1.4.0

- Build first-in-block replay witnesses without debug namespaces using optional
  access-list acceleration, iterative EchoEVM read discovery, EIP-1186 proofs,
  and ordinary RPC lookups.
- Verify account and storage Merkle-Patricia proofs against the parent state
  root and verify fetched bytecode against proved code hashes.
- Preserve optional raw proof bundles and embed up to 256 historical block
  hashes for deterministic `BLOCKHASH` execution.
- Fail closed for later block transactions rather than treating block-boundary
  proofs as intermediate prestate.

## Delivered in v1.3.0

- Add Contract Lens to verified Etherscan address pages using the displayed ABI
  and deployed bytecode without remote code execution or source recompilation.
- ABI-encode and execute functions marked `pure` inside the packaged EchoEVM
  Wasm engine, with decoded output, gas, trace, and bounded causal evidence.
- Keep proxy, storage, external-contract, and Mainnet-state boundaries explicit;
  stateful historical execution continues to require a replay witness.

## Delivered in v1.2.0

- Replace the embedded third-party execution engine with EchoEVM's independent
  Rust opcode, gas, state-transition, call/create, transaction, and precompile
  implementation.
- Preserve the pinned Cancun, Prague, and Osaka zero-skip official fixture gate
  while expanding the owned opcode inventory from 154 to 170 names.

## Delivered in v1.1.0

- Pinned `tests@v20.0.1` official gates for Cancun, Prague, and Osaka: 384 files,
  7,112 transactions, zero skip.
- Machine-readable native/Wasm bytecode contract: 15 exact vectors across 11
  semantic categories and three declared forks, with a frozen 154-opcode
  registration inventory.

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

## Next

- Expand the frozen official corpus to transaction and block fixture families,
  then later execution-spec releases, without shrinking the current gates.
- Grow the bytecode matrix around call/create rollback, warm/cold access,
  precompiles, logs, memory overflow, and transaction-type boundaries.
- Extend proof-backed acquisition to later block transactions by replaying all
  preceding transactions from the proved parent state.
- Add Prague request/system-call and full block transition conformance.
- Add source-level stepping and richer storage/value-flow links while preserving
  the bounded evidence schema.
- Publish the browser extension through a signed store channel after store
  credentials and review are available.

Passing a fixture corpus is a scoped compatibility claim, not proof of full
Ethereum client equivalence.
