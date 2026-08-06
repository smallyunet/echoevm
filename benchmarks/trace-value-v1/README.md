# EchoEVM Trace Value Benchmark

This benchmark isolates the value of the `echoevm.trace.v1` representation for
AI diagnosis. Every condition receives the same bytecode, calldata, execution,
question, model, and response schema. Only the frozen evidence changes:

- `control`: execution result without opcode steps;
- `raw`: Geth-style opcode logs with full pre-op stacks;
- `echo`: bounded explainable EchoEVM events with structured deltas.

The eight cases cover calldata offsets, operand order, memory alignment, return
ranges, hash ranges, storage slots, revert rollback, and invalid jumps. Answers
are graded against exact root-cause, PC, opcode, fix, and secondary-cause fields.
Correctness is primary; tokens and latency are descriptive efficiency metrics.

Generate fixtures from the exact binary under test:

```bash
python3 benchmarks/trace-value-v1/generate_fixtures.py --echoevm ./bin/echoevm
```

Run a smoke test and then the 72-run pilot:

```bash
python3 benchmarks/trace-value-v1/run_benchmark.py \
  --cases storage_rolled_back --repetitions 1 --jobs 2 \
  --output /private/tmp/echoevm-trace-value-smoke

python3 benchmarks/trace-value-v1/run_benchmark.py \
  --repetitions 3 --jobs 2 \
  --output /private/tmp/echoevm-trace-value-pilot

python3 benchmarks/trace-value-v1/analyze_results.py \
  /private/tmp/echoevm-trace-value-pilot
```

The runner preserves prompts, evidence, transcripts, answers, grading, usage,
and versioned fixture metadata. Use task-clustered confidence intervals from
`analysis.json`; do not infer universal Solidity productivity from this bounded
bytecode-diagnosis suite.

Published results:

- [2026-08-06 72-run pilot](RESULTS-2026-08-06.md): semantic accuracy tied at
  100%; full EchoEVM JSON did not beat compact raw opcode evidence on strict
  localization or fresh-token efficiency.
