# Evidence contract

Base conclusions on structured EchoEVM output, not on expected EVM behavior alone.

## Required evidence

Report when available:

- EchoEVM version and `echoevm.evidence.v1` or `echoevm.trace.v1` schema.
- Success, revert, or fault status and return or revert data.
- Total steps plus compact candidate/selected/omitted counts and truncation state; for full traces, matched and emitted counts.
- First relevant step, call depth, PC, opcode, gas, and state delta.
- Persistent/transient storage, memory, stack, and control-flow evidence when selected.
- Causal `enters-frame`, `returns-to`, `rolls-back`, and exact `value-flow` links when selected.
- Declared or transaction fork and replay warnings when applicable.
- Witness schema and SHA-256 provenance for standalone replay.
- Geth version, match fields, and first divergence only when `verify` comparison was requested.

## Interpretation rules

- Label statements directly present in structured output as evidence.
- Label causal explanations, source-level guesses, and suggested fixes as inference.
- One trace applies only to the tested input, environment, gas limit, and initial state.
- `gas.used` on a call/create event can include nested-frame execution; do not label all of it as an opcode surcharge.
- `appliedInFrame` does not mean committed after a later frame or transaction revert.
- When runtime PC mapping is unavailable or ambiguous, identify opcode PC and call depth rather than claiming a Solidity source line.
- Do not market the result as a vulnerability scan, formal verification, or complete compatibility proof.

## Context control

Start with `--format evidence-json --profile auto --limit 40`; route to
`revert`, `storage`, `call`, `abi`, `arithmetic`, or `gas` when the question makes that scope
clear. On truncation or missing context, request an `--around-step` full-trace
window with selected fields and opcode/depth constraints. For full `verify`
comparison output, write JSON to a temporary file and use the compact-result
script. Standalone replay supports bounded evidence directly. Load a full raw trace only when the user
explicitly asks for it or bounded windows cannot answer the question.
