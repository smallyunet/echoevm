# Architecture

EchoEVM keeps stable public entry points in small facade modules and places
implementation details in responsibility-based submodules. The split is
structural: it does not change the protocol, opcode behavior, gas accounting,
fork activation, or fixture expectations.

## Core crate

- `lib.rs` owns the public API and re-exports bytecode, evidence, and replay
  operations from their focused modules.
- `behavior.rs` owns the bounded Behavioral ABI facade; `behavior/` contains
  abstract stack, memory, origin, and effect propagation without changing the
  concrete execution engine.
- `engine.rs` owns execution types and the `Machine` state. Its `engine/`
  modules separate opcode dispatch, instruction helpers, call/create control
  flow, transaction processing, authorization handling, precompiles, gas, and
  arithmetic.
- `tests/official.rs` remains the integration-test entry point. Parsing and
  transaction construction, assertions, and fixture traversal live under
  `tests/official/`.

## CLI crate

- `solidity.rs` remains the command facade. Compilation, ABI coercion, and
  source-map processing live under `solidity/`.
- `witness.rs` owns proof-backed acquisition and shared RPC helpers, while the
  trace-based acquisition adapter lives under `witness/`.

## Maintenance rule

Treat 500 lines as a review signal for hand-written source files, not a blind
limit. Split when a file contains multiple independently testable
responsibilities; keep generated files, lockfiles, tightly coupled tables, and
small cohesive implementations intact. Preserve the facade path when moving an
implementation so downstream callers do not need to change.

Every engine-sensitive structural change must pass the focused bytecode suite,
the workspace checks, and the pinned official fixture gate described in
[`CONFORMANCE.md`](CONFORMANCE.md).
