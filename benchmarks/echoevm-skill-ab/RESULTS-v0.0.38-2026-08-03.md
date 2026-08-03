# EchoEVM v0.0.38 Skill A/B Results — 2026-08-03

## Verdict

The optimized v0.0.38 treatment completed 24 independent Codex runs: four
Solidity tasks, two conditions, and three repetitions. Both conditions passed
all hidden tests (12/12 each). Compared with no skill, the optimized skill
reduced median duration by 18.06%, total tokens by 20.46%, non-cached tokens by
3.80%, output tokens by 9.05%, and command calls from 7 to 6.

This reverses the v0.0.37 pilot's aggregate regressions while retaining
EchoEVM/Geth execution evidence in all 12 treatment finals. The result supports
task-sensitive use of EchoEVM; it does not establish universal savings for all
Solidity work.

## Fixed environment

- EchoEVM: v0.0.38 release-candidate working tree based on `293a6ff`
- Solidity: 0.8.33
- Forge: 1.3.6-nightly
- Codex CLI: 0.146.0-alpha.9.2
- Model: gpt-5.6-sol, medium reasoning
- Runs: 4 tasks x 2 conditions x 3 repetitions = 24
- Ordering: seeded random order, two concurrent workers
- Grading: hidden Foundry tests injected only after each Codex session
- Treatment: repository-local optimized `echoevm-debug` plus pinned v0.0.38 binary
- Control: both EchoEVM skills disabled and EchoEVM replaced with an unavailable executable

All 12 treatment runs read the repository-local skill, verified v0.0.38, passed
their hidden tests, and reported EchoEVM/Geth evidence. All 12 control runs
passed without EchoEVM evidence.

## Aggregate results

| Metric, successful runs only | Control | Optimized skill | Skill delta |
|---|---:|---:|---:|
| Hidden-test pass rate | 12/12 | 12/12 | equal |
| Median duration | 91.60 s | 75.05 s | -18.06% |
| Median input + output tokens | 185,542 | 147,579 | -20.46% |
| Median non-cached input + output tokens | 25,726 | 24,747 | -3.80% |
| Median output tokens | 2,426 | 2,206 | -9.05% |
| Median command calls | 7.0 | 6.0 | -1 call |

`input + output` includes cached input. The non-cached measure is
`input_tokens - cached_input_tokens + output_tokens`. Reasoning tokens are a
subset of output and are not added again.

## Per-task results

| Task | Duration delta | Total-token delta | Non-cached-token delta | Output-token delta |
|---|---:|---:|---:|---:|
| CREATE2 factory | -34.32% | -34.52% | -9.04% | -23.70% |
| Packed decoder | -12.15% | -22.99% | -2.10% | -12.52% |
| Sum-to gas bound | -14.92% | -15.97% | -18.68% | -12.88% |
| Fee quote | +2.14% | -11.20% | +21.39% | +38.75% |

CREATE2 remains the clearest positive case. Separate deployment and runtime gas
limits removed the v0.0.37 sum-to retry loop: its duration delta improved from
+51.40% to -14.92%, and its non-cached-token delta improved from +87.13% to
-18.68%.

Fee quote remains the main routing weakness. The treatment collected several
branch-boundary calls for a high-level arithmetic fix, so fresh-context and
output cost increased even though cached total tokens fell. The next skill
optimization should make the two-execution-call budget explicit and skip
EchoEVM entirely when ordinary source tests cover the behavior.

## Output compaction

For the same matching raw-bytecode differential input, pretty JSON measured
5,587 bytes while `summary-json` measured 1,092 bytes, a reduction of 80.45%.
The summary retains engine versions, status, return data, gas, storage, trace
step counts, match flags, and first-divergence evidence, but omits opcode arrays
and full bytecode. Full traces are now requested only after a divergence.

## v0.0.37 to v0.0.38 change

| Aggregate skill delta | v0.0.37 | v0.0.38 | Improvement |
|---|---:|---:|---:|
| Duration | +14.22% | -18.06% | 32.28 pp |
| Total tokens | +20.25% | -20.46% | 40.71 pp |
| Non-cached tokens | +3.65% | -3.80% | 7.45 pp |
| Output tokens | +33.34% | -9.05% | 42.39 pp |

## Limitations

- The task set is small and all source fixes were solvable without EchoEVM.
- Three repetitions per cell are directional rather than statistically conclusive.
- This compares one model, reasoning level, Codex environment, and machine.
- The v0.0.38 binary was built from the release working tree before its final
  release commit; tagged release CI rebuilds the same tested source at the tag.

Raw valid-run artifacts and the generated `analysis.json` are stored at:

```text
/private/tmp/echoevm-skill-ab-v0038-strict-20260803
```

The earlier same-day run using `~/.pyenv/shims/solc` is excluded because that
shim resolved to an Intel-only solc-select executable and invalidated grading.
