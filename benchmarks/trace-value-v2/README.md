# Solidity causal-evidence benchmark

This benchmark measures whether bounded EchoEVM evidence helps an external
coding agent diagnose real compiled-Solidity executions more accurately and
with less context than a broad opcode trace.

The frozen suite compiles `contracts/NestedFailures.sol` with bundled solc-js
0.8.30 and covers four failure classes: ignored child REVERT, swallowed CREATE
failure, DELEGATECALL storage-context corruption, and a wrong arithmetic
divisor. Each answer must identify the root cause, fix class, and exact primary
and secondary depth/PC/opcode locations.

## Conditions

- `control`: result and task context without opcode evidence.
- `broad`: a broad compact opcode trace.
- `evidence`: question-routed `echoevm.evidence.v1` with causal links.

Generate fixtures with `generate_fixtures.py`, run the external-agent matrix
with `run_benchmark.py`, and summarize it with `analyze_results.py`. The runner
requires an explicitly authorized external `codex exec`; fixture generation
and analysis are local and deterministic.

The release gate requires evidence accuracy to be no worse than broad and the
upper end of the task-clustered 95% confidence interval to show at least 20%
lower fresh-token use. Malformed JSON answers remain failures.

## v0.0.41 formal result

The published `results/2026-08-07-formal` artifact contains all 36 scored runs:
four cases × three conditions × three repetitions, using `gpt-5.6-sol` at
medium reasoning effort. Evidence achieved 11/12 strict answers (91.7%), broad
8/12 (66.7%), and control 0/12. Relative to broad, evidence changed accuracy by
+25 percentage points (95% CI: 0 to +75), fresh tokens by -39.8% (95% CI:
-55.0% to -24.5%), and evidence bytes by -84.1% (95% CI: -91.1% to -78.4%).

This is evidence for these frozen cases and model configuration, not a security
audit, formal verification result, or proof of general EVM compatibility.

## Rust causal-evidence milestone

The fixture generator now executes the current Rust CLI and fails unless all
four cases retain their exact primary and secondary depth/PC/opcode locations.
It additionally requires exact child-failure return and rollback links,
delegate storage-context writes, and the `SUB` PC 591 to `DIV` PC 701 value-flow
carrying divisor `0x3`. Generated Solidity evidence includes the runtime source
map used to enrich those locations.

Each case also has a deterministic `echoevm.explanation.v1` fixture. The gate
requires the generic root-cause class plus both exact oracle locations. Run
`make test-explain` to regenerate the suite in a temporary directory and compare
it byte-for-byte with the checked-in fixtures.

This deterministic fixture gate restores the semantic evidence needed to rerun
the model matrix on Rust. It does not transfer the v0.0.41 model result to the
new implementation; a new external-model result requires a separately
authorized benchmark run.
