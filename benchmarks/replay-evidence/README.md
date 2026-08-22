# Mainnet replay evidence benchmark

This benchmark is the acceptance gate for extending the published Solidity
causal-evidence result to real Ethereum Mainnet transactions. It must use
public, confirmed transactions and frozen replay witnesses so later RPC state,
provider availability, or pruning cannot change the evaluated executions.

## Required case families

- top-level revert with return or custom-error data;
- an internal call failure caught by its caller;
- proxy or `DELEGATECALL` storage-context behavior;
- a provisional storage write rolled back by failure;
- a successful transaction with a non-obvious state transition;
- a dynamic-gas question whose answer is supported by opcode evidence.

Each case must record the transaction hash, block and fork, frozen prestate,
reference opcode trace, post-state diff, question, and a manually reviewed
oracle. The oracle identifies the root cause plus primary and secondary
depth/PC/opcode locations. Fixtures containing credentials, private source, or
non-public transactions are invalid.

## Conditions and gate

Compare a compact receipt/result control, a broad normalized opcode trace, and
question-routed `echoevm.evidence.v1`. Use isolated runs with fixed model,
reasoning effort, prompt, case order seed, and at least three repetitions.
Malformed structured answers remain failures.

The release claim gate requires:

- evidence strict accuracy no worse than broad trace;
- at least 25% fewer fresh tokens at the upper end of the task-clustered 95%
  confidence interval;
- no loss of required root-cause or location fields;
- all cases reproducible without a live RPC after fixture acquisition;
- explicit failure for unsupported fork semantics rather than an inferred
  explanation.

`fresh tokens` means `input_tokens - cached_input_tokens + output_tokens`,
matching the existing Solidity benchmark. Do not publish a Mainnet accuracy or
token-savings number until the complete matrix and auditable artifacts exist.

## Current status

v0.2.0 separates standalone `echoevm.replay-witness.v1` execution from optional
debug-RPC import and Geth verification. No external-model Mainnet result is
claimed yet because the complete public frozen-witness matrix has not been run
with authorization to submit its evidence to an external model.
