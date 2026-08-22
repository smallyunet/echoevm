# EchoEVM protocol v1 freeze

This directory freezes the public compatibility contract inherited from the Go
implementation at commit `327afd6d52774b30f5ac022bc3f5668e44825f7f`
(`v0.4.1`). The Rust implementation may add fields, commands, and formats, but
must not remove or reinterpret the contracts below without introducing a new
schema version.

## Versioned JSON contracts

- `echoevm.trace.v1` is the opcode trace document and JSONL event schema.
- `echoevm.evidence.v1` is the bounded causal-evidence document schema.
- `echoevm.replay-witness.v1` is the strict, self-contained transaction replay
  input. Unknown fields are rejected, input is limited to 64 MiB, and one file
  contains exactly one JSON document.
- Solidity editor responses use numeric `schemaVersion: 1` for inspect, run,
  summary, and error envelopes.

JSON property names, hexadecimal encoding, status values (`success`, `revert`,
and `fault`), omission rules, and the distinction between execution completeness
and presentation truncation are normative. Object property order and whitespace
are not normative.

## Stable commands

The Rust CLI must retain these top-level commands and their v1 meanings:

`call`, `deploy`, `disasm`, `repl`, `replay`, `run`, `solidity inspect`,
`solidity run`, `trace`, `version`, `web`, and `witness import-debug`.

The stable machine-facing formats are `json`, `jsonl`, `summary-json`, and
`evidence-json`. Human-readable text may improve without a protocol bump.

## Execution contract

- Standalone execution is performed by EchoEVM. A foreign execution engine is
  never a runtime backend.
- Replay consumes only a complete witness and does not contact RPC.
- `witness import-debug` is an optional acquisition adapter; its output must
  replay offline.
- Cancun through Osaka are the declared transaction/interpreter rulesets.
- Pre-Cancun replay, proof-backed historical state acquisition, block-level
  system processing, and fixture families not executed by the release gate stay
  outside the compatibility claim.
- `--limit` bounds emitted evidence, not execution. Counts and `truncated` must
  continue to distinguish full execution from partial presentation.

## Release gate

Rust becomes the default implementation only after it passes all existing
protocol consumers and, at minimum, the pinned `tests@v20.0.1` executable corpus:

- 187 Prague/Osaka-authored state fixture files;
- 3,461 transactions;
- 3,244 accepted transactions;
- 217 consensus-invalid transactions rejected;
- zero skipped execution.

Independent regression vectors and curated compliance fixtures must be
non-shrinking and zero-skip. Official fixtures, not Go output or another client,
remain the semantic oracle. The Go implementation is retained on the `go`
branch as a compatibility reference, not a backend.

## Normative source snapshot

The exact v1 field definitions remain recoverable from the frozen `go` branch:

- `internal/trace/model.go` and `internal/trace/evidence.go`;
- `internal/replay/witness.go` and `internal/replay/model.go`;
- `cmd/echoevm/solidity.go` and `cmd/echoevm/summary.go`;
- `editors/vscode/src/protocol.ts`;
- `docs/TRACE_PROTOCOL.md` and `docs/REPLAY_WITNESS.md`.

Rust compatibility tests must encode these contracts independently so deleting
or changing the Go implementation cannot silently change the protocol.
