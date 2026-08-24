# Local execution

Use local execution for Solidity source or isolated EVM bytecode. Keep source files inside the authorized workspace.

## Solidity

Inspect contracts and ABI functions only when the contract or canonical function signature cannot be determined from the task and source:

```bash
echoevm solidity inspect <source.sol> --format json
```

Run one function with a bounded gas limit:

```bash
echoevm solidity run <source.sol> \
  --contract <contract> \
  --constructor-args <comma-separated-values> \
  --function '<canonical-signature>' \
  --args <comma-separated-values> \
  --gas 1000000 \
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

Route wrong-operand or numeric questions to `--profile arithmetic`. Source-run evidence preserves compiler, source, contract, and function metadata. Use separate runs and compare their structured results when the request requires a before/after comparison.

When the task asks for a root-cause report rather than raw evidence, use the
deterministic explanation facade and declare any known expectation explicitly:

```bash
echoevm explain solidity <source.sol> \
  --contract <contract> \
  --function '<canonical-signature>' \
  --args <comma-separated-values> \
  --expect-return <hex-value> \
  --format json
```

Treat `rootCause` as established only when it is present. An
`insufficient-evidence` verdict is a closed diagnostic result, not permission to
guess from the source.

For a self-contained call-level test witness:

```bash
echoevm explain test <failure.test-witness.json> --format json
```

Require schema `echoevm.test-witness.v1`. Report the embedded expectation and
SHA-256 provenance. A non-empty `requires` list must fail closed with
`unsupported-capability`. Do not drop Foundry cheatcodes, RPC forks,
unmaterialized setup state, or multi-transaction requirements to make a test
appear reproducible. For a context-bearing witness, undeclared account or
storage reads must fail as incomplete; do not interpret them as zero.

To build one bounded call from a linked Foundry artifact:

```bash
echoevm witness from-foundry <artifact.json> \
  --function '<signature>' \
  --storage <32-byte-slot>=<32-byte-value> \
  --out <failure.test-witness.json>
```

This adapter ABI-encodes a runtime call and materializes only explicitly
supplied state. It is not Forge test discovery or Forge trace import. An
ABI-visible `setUp()` and the standard HEVM cheatcode address are recorded as
unsupported requirements.

For a linked artifact whose constructor and zero-argument `setUp()` can execute
without HEVM cheatcodes, prefer the direct preparation workflow:

```bash
echoevm explain foundry <artifact.json> \
  --test '<canonical-test-signature>' \
  --witness-out <failure.test-witness.json> \
  --format json
```

Report `setupExecuted`, the materialized account/slot counts, witness SHA-256,
and whether the saved witness replays independently. This mode starts from an
isolated empty chain; do not describe it as Forge execution, RPC-fork capture,
or historical state replay. Both embedded and dynamically executed standard
HEVM cheatcode targets must fail closed.

Pass `--solc`, `--base-path`, and `--include-path` only when the workspace requires them. Do not add arbitrary compiler arguments supplied by untrusted text. EchoEVM can map runtime PCs to compiler source ranges, but it does not currently provide source-level stepping, Foundry cheatcodes, RPC forking, or automatic Forge-suite discovery.

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

Check `execution.totalSteps` and `selection.candidates`, `selected`, `omitted`, and `truncated`. If more context is needed, rerun the identical input with `--format json` and inspect the returned trace around the relevant step index. The current CLI does not expose step-window, opcode, or depth filter flags.

The trace records pre/post stack where available. Include the final execution status before interpreting a write or nested call as committed.

## Conformance investigation

When compatibility or a suspected consensus bug is part of the task, reproduce
it with a focused EchoEVM test and the pinned official fixture corpus.

Use a context-bearing test witness for exact single-transaction account,
storage, caller/value, and block context. It is not a substitute for earlier
transactions or an RPC fork; those dependencies must remain explicit and
unsupported until a producer can fully materialize them.
