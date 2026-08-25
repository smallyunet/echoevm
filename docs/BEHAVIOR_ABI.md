# Behavioral ABI

EchoEVM can infer a bounded behavioral summary directly from deployed runtime
bytecode. It does not require Solidity source, contract verification, an ABI,
RPC access, or a concrete transaction:

```bash
echoevm behavior 600035631122334414600d57005b60043560015500 --format json
```

The result uses `echoevm.behavior.v1`. It identifies selectors from common EVM
dispatchers, explores statically recoverable control flow, and reports tracked
effects for persistent and transient storage, logs, account/code reads,
external calls, delegate execution, contract creation, and self-destruction.

## Value origins

The abstract executor retains compact origins when it can establish them, such
as `caller`, `callvalue`, `calldata.arg0`, `storage[constant(0x1)]`, or a bounded
expression combining those values. An origin of `unknown` is retained rather
than guessed.

Each function summary includes:

- the four-byte selector and optional ABI signature supplied by a frontend;
- the recovered entry program counter;
- effect-derived capability labels;
- effect locations and recovered inputs;
- caller, storage, value, or environment-dependent branch conditions; and
- per-entry reachability, unresolved-jump, and truncation coverage.

The document also contains contract-wide unions plus bytecode SHA-256
provenance. ABI signatures are labels only: they do not change inference.

## Chrome Behavior Lens

On an Etherscan address page the Chrome extension reads the deployed bytecode
already rendered by the page and automatically invokes the same Rust analyzer
through WebAssembly. A verified ABI, when present, supplies human-readable
function signatures. Bytecode analysis remains available without runnable
`pure` functions.

No Etherscan API, RPC endpoint, remote executor, or uploaded contract data is
used. The extension requests no broad host or network permissions beyond its
declared Etherscan content-script matches.

## Boundary

Behavioral ABI is bounded abstract interpretation, not concrete execution,
decompilation, a security finding, or formal verification. In particular:

- selector recovery currently recognizes common `PUSH4`/`EQ` dispatchers;
- fallback, EOF, computed dispatch, and some compiler-specific paths may be
  absent from per-function summaries;
- dynamic jumps and external call destinations remain unresolved when their
  values cannot be recovered;
- paths are bounded to prevent loops or branching from exhausting the browser;
- a reported capability means a tracked effect is reachable in the recovered
  abstract control flow, not that every caller can necessarily exercise it;
- a missing effect is not proof that the behavior is impossible.

Use a self-contained test or replay witness when a possible behavior must be
confirmed by concrete EchoEVM execution.
