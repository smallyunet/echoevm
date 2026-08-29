# EchoEVM roadmap

**Current release: v1.8.0 — self-contained block execution and arbitrary-position replay**

## Delivered in v1.8.0

- Execute strict `echoevm.block-witness.v1` inputs sequentially with one shared
  prestate and verify the header hash, transaction root, withdrawals root, gas
  used, receipts root, logs bloom, and final state root.
- Apply Cancun beacon-root processing and Prague/Osaka history, withdrawal
  request, consolidation request, and withdrawal transitions without using a
  foreign execution backend.
- Extend standard-RPC `witness import-proof` to arbitrary transaction positions
  by proving parent state and locally replaying every preceding transaction to
  materialize the exact intermediate target prestate.
- Add pinned `tests@v20.0.1` gates for 41,922 accepted single-block transitions
  and 113 declared-invalid transaction fixtures across Cancun, Prague, and Osaka,
  while preserving the 39,921-state-test transaction gate.
- Improve Behavioral ABI fallback inference and make Chrome Behavior Lens
  injection explicit and test-covered.

## Delivered in v1.7.0

- Infer `echoevm.behavior.v1` directly from runtime bytecode without Solidity
  source, an ABI, RPC access, or a concrete transaction.
- Recover common selectors, statically reachable state/call/create/log effects,
  compact value origins, semantic branch conditions, and explicit coverage
  limits through bounded abstract execution.
- Add `echoevm behavior` JSON/text output and the same Rust analysis through
  WebAssembly.
- Make Chrome Behavior Lens automatically recognize deployed bytecode rendered
  on Etherscan contract pages; use a verified ABI only to label selectors and
  preserve the existing optional pure-function sandbox.
- Keep inferred capability distinct from exploitability, concrete execution,
  decompilation, auditing, and formal proof.

## Delivered in v1.6.0

- Add direct `echoevm explain foundry` preparation for linked artifacts:
  constructor deployment, zero-argument `setUp()`, explicit read-set closure,
  independently replayable witness output, and deterministic SHA-256
  provenance.
- Carry explicit accounts, storage, caller, target, value, and block context in
  strict call-level test witnesses while failing closed on undeclared reads.
- Track storage values through stack and memory into RETURN/REVERT so return
  mismatches can identify `storage-output-provenance` as direct root evidence.
- Reject standard and dynamically executed HEVM cheatcodes rather than treating
  Forge behavior as ordinary empty-account calls.
- Keep the scope to a fresh isolated single-test chain; RPC forks, external
  historical state, and Forge orchestration remain outside the claim.

## Delivered in v1.5.1

- Split the execution engine, transaction processing, bytecode, replay,
  evidence, official fixture runner, Solidity tooling, and witness acquisition
  into responsibility-based modules while preserving public API paths.
- Keep every hand-written Rust, TypeScript, and JavaScript source file at or
  below the 500-line maintenance review threshold.
- Preserve the exact 39,921-transaction Cancun-through-Osaka official fixture
  gate, native/Wasm bytecode vectors, protocol v1, and release artifact matrix.
- Clarify that exact traces are the execution record and bounded evidence is a
  deterministic diagnostic selection, not an inferred causal graph.

## Delivered in v1.5.0

- Expand the pinned `tests@v20.0.1` gate to every matching Cancun, Prague, and
  Osaka state-test fixture: 7,216 files and 39,921 transactions, zero skip.
- Verify signed transaction round-trip and sender, normalized rejection class,
  receipt gas/status, logs commitment, account state, and state root.
- Add EIP-152 BLAKE2F, high-s ECRECOVER handling, EIP-2200 SSTORE sentry,
  fork-aware blob limits, EIP-7610 storage collisions, and the edge semantics
  exposed by the expanded corpus.
- Expose execution logs, logs hash, and state root in the additive v1 result.

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
  Wasm engine, with decoded output, gas, trace, and bounded execution evidence.
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

- Execute and classify rejected and multi-block blockchain fixtures, then track
  later execution-spec releases without shrinking any current gate.
- Grow the bytecode matrix around call/create rollback, warm/cold access,
  precompiles, logs, memory overflow, and transaction-type boundaries.
- Recompute and verify Prague/Osaka `requestsHash` from the emitted request list.
- Add source-level stepping and richer branch/control value-flow links while
  preserving the bounded evidence schema.
- Publish the browser extension through a signed store channel after store
  credentials and review are available.

Passing a fixture corpus is a scoped compatibility claim, not proof of full
Ethereum client equivalence.
