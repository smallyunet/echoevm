# Evidence contract

Base conclusions on structured EchoEVM output, not on expected EVM behavior alone.

## Required evidence

Report when available:

- EchoEVM version and `echoevm.explanation.v1`, `echoevm.evidence.v1`, or `echoevm.trace.v1` schema.
- Success, revert, or fault status and return or revert data.
- Total steps plus compact candidate/selected/omitted counts and truncation state; for full traces, matched and emitted counts.
- First relevant step, call depth, PC, opcode, gas, and stack transition.
- Stack, storage/context, and control-flow evidence present in selected steps.
- Causal `enters-frame`, `returns-to`, `rolls-back`, and tracked arithmetic `value-flow` links when selected.
- Explanation verdict, optional root cause, direct findings, and limitations when `echoevm explain` is used.
- Declared or transaction fork and replay warnings when applicable.
- Witness schema and SHA-256 provenance for standalone replay.
- Pinned fixture release, fork, exact case count, and failure evidence when conformance was requested.

## Interpretation rules

- Label statements directly present in structured output as evidence.
- Label causal explanations, source-level guesses, and suggested fixes as inference.
- One trace applies only to the tested input, environment, gas limit, and initial state.
- `gas.used` on a call/create event can include nested-frame execution; do not label all of it as an opcode surcharge.
- Treat a `value-flow` link as tracked stack provenance for that execution, not a source-level business invariant.
- When runtime PC mapping is unavailable or ambiguous, identify opcode PC and call depth rather than claiming a Solidity source line.
- Do not market the result as a vulnerability scan, formal verification, or complete compatibility proof.

## Context control

Start with `--format evidence-json --profile auto --limit 40`; route to `revert`, `storage`, `call`, `abi`, `arithmetic`, or `gas` when the question makes that scope clear. On truncation or missing context, rerun the identical input with `--format json` and inspect the smallest relevant slice of the returned trace. Standalone replay supports bounded evidence directly.
