# Evidence contract

Base conclusions on structured EchoEVM output, not on expected EVM behavior alone.

## Required evidence

Report when available:

- EchoEVM version and `echoevm.trace.v1` schema.
- Success, revert, or fault status and return or revert data.
- Total, matched, and emitted step counts plus truncation state.
- First relevant step, call depth, PC, opcode, gas, and state delta.
- Persistent/transient storage, memory, stack, and control-flow evidence when selected.
- Declared or transaction fork and replay warnings when applicable.
- Geth version, match fields, and first divergence only when a comparison was requested.

## Interpretation rules

- Label statements directly present in structured output as evidence.
- Label causal explanations, source-level guesses, and suggested fixes as inference.
- One trace applies only to the tested input, environment, gas limit, and initial state.
- `gas.used` on a call/create event can include nested-frame execution; do not label all of it as an opcode surcharge.
- `appliedInFrame` does not mean committed after a later frame or transaction revert.
- Without source maps, identify opcode PC and call depth, not a Solidity source line.
- Do not market the result as a vulnerability scan, formal verification, or complete compatibility proof.

## Context control

Use explainable trace filters before reading output: changes-only, selected fields,
opcode/depth constraints, and a reasonable limit. On truncation, request an
`--around-step` window. For legacy comparison or replay output, write full JSON
to a temporary file and use the compact-result script. Load a full raw trace only
when the user explicitly asks for it or bounded windows cannot answer the question.
