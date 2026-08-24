# Trace and evidence protocol

EchoEVM executes the complete input before formatting output. `--limit` changes
presentation only and never stops execution.

## Exact trace

`echoevm.trace.v1` steps contain the global index, call depth, program counter,
opcode byte/name, gas before/after, pre-op stack, optional post-op stack, and an
optional halt/error classification. `trace --format jsonl` emits one JSON step
per line; `json` emits the complete execution envelope; `text` is for humans.

```bash
echoevm trace 600160020100 --format jsonl
```

## Bounded evidence

`--format evidence-json` emits `echoevm.evidence.v1`. Selection happens after
the exact trace is complete. `selection.candidates`, `selected`, `omitted`, and
`truncated` state the evidence boundary explicitly.

Profiles are `auto`, `revert`, `storage`, `call`, `abi`, `gas`, `arithmetic`,
and `full`. They route presentation toward a question without changing EVM
semantics.

```bash
echoevm trace 600160020100 \
  --format evidence-json --profile arithmetic --limit 20

echoevm solidity run ./Contract.sol \
  --contract Contract --function 'run(uint256)' --args 42 \
  --format evidence-json --profile auto --limit 40

echoevm replay ./transaction.witness.json \
  --format evidence-json --profile revert --limit 40
```

Evidence explanations are deterministic and derived from executed opcodes.
The Rust selector emits causal `links` only when the captured trace establishes
them: call/create frame entry and return, rolled-back storage writes, and
tracked arithmetic value flow. Storage events include their execution-context
address and observed previous/value pair. Selected Solidity evidence events
also carry source locations resolved from the runtime source map when available.

## Deterministic explanation

`echoevm explain replay` and `echoevm explain solidity` consume the same bounded
evidence and emit `echoevm.explanation.v1`. The document separates `verdict`, an
optional `rootCause`, all captured `findings`, execution and evidence summaries,
and explicit `limitations`.

Declared `--expect-status` and `--expect-return` values establish a mismatch;
they do not themselves establish its cause. A root cause is emitted only when
the selected frame, rollback, storage-context, or arithmetic provenance links
close the causal chain. Otherwise the verdict is `insufficient-evidence`.

When `--expect-return` is present and the profile remains `auto`, explain routes
to the arithmetic profile. A short expected ABI word is compared with its
32-byte zero-padded return encoding; arbitrary return byte strings otherwise
remain exact.

`echoevm explain test <witness>` accepts `echoevm.test-witness.v1`. The embedded
expectation can cover status, return data, and selected final storage slots.
Test witnesses with return expectations route to arithmetic evidence; storage
expectations route to storage evidence; combined expectations route to full
evidence. Optional PC source locations are copied only onto matching events.

## Boundary

Trace and evidence are produced by the embedded Rust executor. They do not
contact an RPC and are not comparisons with another client. Transaction replay
uses only the supplied witness and supports the declared Cancun-through-Osaka
transaction/interpreter scope.
