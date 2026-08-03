# EchoEVM Skill A/B Benchmark

This benchmark compares Codex Solidity work with and without the
`echoevm-debug` skill. Correctness is graded before time or token efficiency.

The pilot contains four tasks and runs each condition three times. Public task
files are copied into isolated temporary Git repositories. Hidden Foundry tests
are injected only after each Codex session finishes.

Conditions:

- `control`: disables both EchoEVM skills and shadows `echoevm` with an
  unavailable executable.
- `skill`: enables `echoevm-debug`, disables `echoevm-conformance`, and exposes
  the pinned benchmark binary as `echoevm`.

Every run receives the same `AGENTS.md`, which requires explicit
`./.benchmark-bin/...` paths. This prevents login-shell PATH changes from
silently selecting a different installed EchoEVM, solc, or Forge version.

The runner stores every prompt, JSONL transcript, patch, final response,
grading log, and aggregate result under the selected output directory.

Generate a per-task analysis and evidence audit after a run:

```bash
python3 benchmarks/echoevm-skill-ab/analyze_results.py \
  /private/tmp/echoevm-skill-ab-results
```

Run the full pilot:

```bash
python3 benchmarks/echoevm-skill-ab/run_benchmark.py \
  --echoevm ./bin/echoevm \
  --solc ~/.svm/0.8.33/solc-0.8.33 \
  --forge ~/.foundry/bin/forge \
  --repetitions 3 \
  --jobs 2
```

Run a smoke sample first:

```bash
python3 benchmarks/echoevm-skill-ab/run_benchmark.py \
  --echoevm ./bin/echoevm \
  --solc ~/.svm/0.8.33/solc-0.8.33 \
  --forge ~/.foundry/bin/forge \
  --repetitions 1 \
  --tasks fee_quote \
  --conditions control,skill
```
