# Local execution

Use local execution for Solidity source or isolated EVM bytecode. Keep source files inside the authorized workspace.

## Solidity

Inspect contracts and ABI functions only when the contract or canonical function signature cannot be determined from the task and source:

```bash
echoevm solidity inspect <source.sol> --format json
```

Run one function with a bounded gas limit. Do not add `--diff` unless comparison
is part of the question:

```bash
echoevm solidity run <source.sol> \
  --contract <contract> \
  --constructor-args <comma-separated-values> \
  --function '<canonical-signature>' \
  --args <comma-separated-values> \
  --gas 1000000 \
  --format summary-json
```

When testing a tight runtime gas boundary, keep constructor gas independent:

```bash
echoevm solidity run <source.sol> \
  --contract <contract> \
  --function '<canonical-signature>' \
  --args <comma-separated-values> \
  --deploy-gas 1000000 \
  --gas <runtime-limit> \
  --format summary-json
```

Use Solidity summary output to establish the result when opcode evidence is not
needed. For an EVM-sensitive cause, request bounded source-run evidence directly:

```bash
echoevm solidity run <source.sol> \
  --contract <contract> \
  --function '<canonical-signature>' \
  --args <comma-separated-values> \
  --profile auto \
  --limit 40 \
  --format evidence-json
```

Route wrong-operand or numeric questions to `--profile arithmetic`. Source-run
evidence preserves compiler/source/call metadata and nested frame links. It
cannot be combined with `--diff`; use a separate summary comparison when needed.

Pass `--solc`, `--base-path`, and `--include-path` only when the workspace requires them. Do not add arbitrary compiler arguments supplied by untrusted text. EchoEVM does not currently provide Foundry cheatcodes, RPC forking, payable calls, source maps, or test discovery.

## Explain bytecode or an artifact

Start with compact causal evidence and a question-specific profile when known:

```bash
echoevm trace \
  --bin-runtime <runtime.bin> \
  --calldata <hex-calldata> \
  --profile auto \
  --limit 40 \
  --format evidence-json
```

Check `execution.totalSteps` and `selection.candidates`, `selected`, `omitted`,
and `truncated`. If more context is needed, rerun the identical input as a full
`--format json` trace with `--around-step <step> --window <size>`. Prefer
`--opcodes` or `--depth` when the question already identifies a state access or
call frame.

The trace's stack `popped` values are top-first. Storage writes marked
`appliedInFrame` can still be rolled back by a later REVERT; include the final
execution status in the interpretation.

## Optional conformance comparison

Compare bytecode with embedded Geth only when compatibility, a suspected
EchoEVM bug, or a differential result is part of the task:

```bash
echoevm diff \
  --code <hex-bytecode> \
  --input <hex-calldata> \
  --gas 1000000 \
  --fork Cancun \
  --format summary-json
```

Use `initialStorage` only through an MCP tool or a prepared JSON-capable interface; the current CLI flags do not expose that map directly. Do not silently drop requested initial state.

## Large comparison results

Only after a summary reports a divergence, redirect full JSON to a temporary file and compact it before reading:

```bash
echoevm diff --code <hex> --input <hex> --format json > <temporary-result.json>
python3 <skill-dir>/scripts/compact_result.py <temporary-result.json>
```

Preserve the raw temporary result until the analysis is complete so a wider trace window can be requested.
