# Evidence contract

Base conclusions on structured EchoEVM output, not on expected EVM behavior alone.

## Required evidence

Report when available:

- EchoEVM version and Geth module version.
- Declared or transaction fork.
- Success, revert, or fault status for each engine.
- Return or revert data comparison.
- Gas comparison.
- Persistent storage or replay post-state comparison.
- Trace lengths and trace match.
- First divergence step, PC, opcode, field, and both values.
- Replay warnings.

## Interpretation rules

- Label statements directly present in structured output as evidence.
- Label causal explanations, source-level guesses, and suggested fixes as inference.
- A `MATCH` applies only to the tested input, environment, gas limit, and initial state.
- A trace mismatch can precede a matching final result; report both facts.
- Nested calls and creates can contain trace or gas fields that are intentionally not comparable. Preserve `traceSemantics` instead of inventing a comparison.
- Without source maps, identify opcode PC and call depth, not a Solidity source line.
- Do not market the result as a vulnerability scan, formal verification, or complete compatibility proof.

## Context control

Use `summary-json` by default. On divergence, write full JSON to a temporary file and use the compact-result script; matching traces retain counts but no opcode steps. If more context is required, rerun the compactor with `--window <steps>` up to a reasonable bound. Load the full raw trace only when the user explicitly asks for it or a compact window cannot answer the question.
