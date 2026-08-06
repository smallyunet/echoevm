# EchoEVM Trace Value Benchmark

This benchmark isolates the value of EchoEVM trace representations for
AI diagnosis. Every condition receives the same bytecode, calldata, execution,
question, model, and response schema. Only the frozen evidence changes:

- `control`: execution result without opcode steps;
- `raw`: Geth-style opcode logs with full pre-op stacks;
- `echo`: bounded explainable EchoEVM events with structured deltas.
- `evidence`: compact causal `echoevm.evidence.v1` selected with `auto`.

The eight base cases cover calldata offsets, operand order, memory alignment, return
ranges, hash ranges, storage slots, revert rollback, and invalid jumps. Answers
are graded against exact root-cause, PC, opcode, fix, and secondary-cause fields.
Correctness is primary; tokens and latency are descriptive efficiency metrics.

Generate fixtures from the exact binary under test:

```bash
python3 benchmarks/trace-value-v1/generate_fixtures.py --echoevm ./bin/echoevm
```

Run a smoke test and then the 96-run four-condition matrix:

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

Four explicit `*_noise96` variants prepend semantically neutral stack setup to
measure representation scaling. Run the 24-run raw/evidence gate with:

```bash
python3 benchmarks/trace-value-v1/run_benchmark.py \
  --cases calldata_selector_offset_noise96,division_operand_order_noise96,storage_wrong_slot_noise96,storage_rolled_back_noise96 \
  --conditions raw,evidence --repetitions 3 --jobs 4 \
  --output /private/tmp/echoevm-trace-value-scale
```

The runner preserves prompts, evidence, transcripts, answers, grading, usage,
and versioned fixture metadata. Use task-clustered confidence intervals from
`analysis.json`; do not infer universal Solidity productivity from this bounded
bytecode-diagnosis suite.

Published results:

- [2026-08-06 72-run pilot](RESULTS-2026-08-06.md): semantic accuracy tied at
  100%; full EchoEVM JSON did not beat compact raw opcode evidence on strict
  localization or fresh-token efficiency.
- [2026-08-06 compact evidence validation](RESULTS-2026-08-06-EVIDENCE.md): the
  96-run base matrix preserved correctness, and the 24-run scaling gate reduced
  fresh tokens by 34.6% with a task-clustered 95% interval excluding 20%.
