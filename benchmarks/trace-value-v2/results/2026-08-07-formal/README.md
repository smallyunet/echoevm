# Formal result: 2026-08-07

- Matrix: 4 compiled-Solidity cases × 3 conditions × 3 repetitions = 36 runs
- Agent: `gpt-5.6-sol`, medium reasoning effort
- Evidence: 11/12 strict answers, median score 10, median fresh tokens 16,158.5
- Broad: 8/12 strict answers, median score 10, median fresh tokens 27,312
- Control: 0/12 strict answers, median score 6, median fresh tokens 14,858
- Evidence vs broad fresh tokens: -39.8% (task-clustered 95% CI -55.0% to -24.5%)
- Evidence vs broad accuracy: +25 percentage points (95% CI 0 to +75)

Three zero-score outputs were retained: each was a completed model response
whose `ANSWER.json` omitted its final closing brace; the strict JSON contract
therefore rejected it. No infrastructure-invalid run was excluded or replaced.

See `analysis.json` for aggregate statistics, `runs.jsonl` for per-run scores
and usage, and `run-metadata.json` for the frozen environment.
