# Compact Evidence Validation — 2026-08-06

## Verdict

`echoevm.evidence.v1` passes the release gate for long traces while preserving
diagnostic correctness on the original small-task matrix. It should be used as
the default agent-facing trace view; the full `echoevm.trace.v1` document remains
available for deterministic follow-up windows and protocol-level inspection.

This result is bounded evidence for bytecode diagnosis, not a claim about all
Solidity debugging or complete EVM compatibility.

## Base matrix

The locked matrix used eight cases, four frozen-evidence conditions, three
repetitions, `gpt-5.6-sol` at medium reasoning, seed 20260806, and four workers.
All 96 processes produced valid answers.

| Metric | Control | Raw | Full Echo | Compact evidence |
|---|---:|---:|---:|---:|
| Strict diagnosis | 23/24 | 24/24 | 23/24 | 24/24 |
| Causal diagnosis | 24/24 | 24/24 | 24/24 | 24/24 |
| Median fresh tokens | 15,223.5 | 14,654 | 16,418 | 13,629.5 |
| Median evidence bytes | 386 | 973.5 | 3,448.5 | 879 |

The task-clustered evidence-vs-raw fresh-token estimate was noisy on these tiny
executions: +8.2%, 95% CI [-7.6%, +25.0%]. Therefore the small-task median was
not used to claim efficiency. Correctness and causal accuracy showed no
regression.

## Long-trace scaling gate

Four base cases were expanded with 96 neutral `PUSH1 0; POP` pairs before the
failing logic. Both conditions received identical bytecode, prompt, model, and
oracle; only raw versus compact evidence differed. All 24 processes were valid.

| Metric | Raw | Compact evidence |
|---|---:|---:|
| Strict diagnosis | 12/12 | 12/12 |
| Causal diagnosis | 12/12 | 12/12 |
| Median fresh tokens | 25,341 | 15,486 |
| Median duration | 31.302 s | 30.453 s |
| Median evidence bytes | 15,714 | 1,623 |

Task-clustered relative deltas for compact evidence versus raw were:

- fresh tokens: **-34.6%**, 95% CI **[-43.0%, -21.3%]**;
- evidence bytes: **-89.8%**, 95% CI **[-90.8%, -88.9%]**;
- duration: -4.5%, 95% CI [-16.1%, +3.6%].

The predeclared efficiency gate required equal correctness with at least 20%
lower fresh-token use. The complete fresh-token confidence interval clears that
threshold. The original eight cases serve as short-trace correctness controls;
they show no strict or causal regression against raw evidence.

Auditable aggregate analyses, run metadata, answers, grading, and usage are in
`results/2026-08-06-evidence-matrix/` and
`results/2026-08-06-evidence-scale/`. Full transcripts remain in the raw
artifact directories recorded by the run metadata.
