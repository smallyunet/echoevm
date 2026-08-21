# Explainable trace protocol

EchoEVM's primary trace interface is an AI-oriented explanation of one EVM
execution. It describes what each opcode changed and why the step matters. A
Geth comparison is a separate conformance operation, not a prerequisite for
using the trace.

## Schema

The current schema identifier is `echoevm.trace.v1`. `--format json` emits one
document; `--format jsonl` emits one `opcode` record per selected event followed
by one `result` record. Consumers should route on both `type` and `schema` and
ignore unknown fields.

Each opcode event always includes:

- global `step`, call `depth`, executing `address`, `pc`, opcode byte, and name;
- halt, revert, and execution-error state when applicable.

Optional event fields are selected with `--fields`:

- `gas`: remaining gas before and after the opcode, net gas used, static cost,
  and the remaining additional cost. For call/create opcodes, net gas used can
  include child-frame execution.
- `stack`: words popped in top-first order and words pushed in bottom-to-top
  order, plus sizes before and after. SWAP opcodes are represented explicitly
  as a top-first reordering rather than as fake pop/push operations.
- `memory`: size changes and changed byte ranges. `--max-memory-bytes` bounds
  the bytes retained for a single event and marks a clipped delta.
- `storage`: persistent or transient reads and writes, slot values, warm/cold
  status, original value when relevant, and whether a write executed in its
  frame. A later frame or transaction revert can still roll that write back.
- `control`: jump, call, create, return, revert, or stop behavior.
- `explanation`: a deterministic sentence derived from the structured fields.

The final execution record distinguishes:

- `totalSteps`: all opcodes executed;
- `matchedSteps`: events selected by range, depth, opcode, and change filters;
- `emittedSteps`: matched events retained after `--limit`;
- `filtered`: selection excluded some executed events;
- `truncated`: `--limit` clipped matching events.

`--limit` never stops EVM execution.

### Compact causal evidence

`--format evidence-json` emits `echoevm.evidence.v1`, a compact view selected
after the complete execution has finished. It preserves execution status and
prioritizes faults, reverts, storage, calls/creates, returns, memory changes,
and semantic opcodes. Evidence can also include causal links: `enters-frame`
and `returns-to` connect parent and child execution, `rolls-back` identifies
provisional state discarded by failure, and `value-flow` links an exact stack
producer to its consumer through stack duplication and reordering. The default
`auto` profile omits stack-only plumbing such
as PUSH, DUP, SWAP, POP, and JUMPDEST unless a higher-priority event requires it.

Profiles route common questions without changing execution:

- `auto`: general causal evidence;
- `revert`: revert data, memory, state, and surrounding call/return control;
- `storage`: persistent/transient access and commit/rollback control;
- `call`: call/create control and nested-frame semantic events;
- `abi`: calldata, memory, hashing, return, and revert evidence;
- `arithmetic`: arithmetic consumers and the producers of their operands;
- `gas`: the auto selection with gas deltas retained;
- `full`: every selected opcode, compactly encoded.

`selection.candidates`, `selected`, `omitted`, and `truncated` make the evidence
boundary explicit. With evidence JSON, `--limit` is applied after full execution
and priority selection, so a late terminal fault is not lost to an early limit.

## Agent workflow

Start with compact causal evidence:

```bash
echoevm trace \
  --bin-runtime ./runtime.bin \
  --calldata 0x1234 \
  --profile auto \
  --limit 40 \
  --format evidence-json
```

For Solidity source, the same evidence contract includes compilation and call
metadata while preserving the complete deploy-then-call execution result:

```bash
echoevm solidity run ./Contract.sol \
  --contract Contract \
  --function 'run(uint256)' \
  --args 42 \
  --profile auto \
  --limit 40 \
  --format evidence-json
```

`solidity run --format evidence-json` is an EchoEVM execution view. It cannot be
combined with `--diff`; request `summary-json --diff` separately when an engine
comparison is required.

For a confirmed Ethereum Mainnet transaction, replay evidence adds immutable
transaction and fork provenance, compatibility warnings, and a compact
comparison verdict to the same evidence contract:

```bash
echoevm replay 0x0123... \
  --profile revert \
  --limit 40 \
  --format evidence-json
```

The replay evidence envelope deliberately omits the duplicate full EchoEVM and
Geth traces. Use `replay --format json` only when the complete differential
result or a wider comparison window is required.

If the selection is truncated or a step needs more context, request a
deterministic full-trace window without rerunning a different input:

```bash
echoevm trace \
  --bin-runtime ./runtime.bin \
  --calldata 0x1234 \
  --around-step 42 \
  --window 5 \
  --format json
```

Opcode and call-frame filters can isolate state access or nested execution:

```bash
echoevm trace -r ./runtime.bin -d 0x1234 \
  --opcodes SLOAD,SSTORE,TLOAD,TSTORE --depth 1 --format jsonl
```

## Current boundary

The explainable protocol fronts the standalone `trace` command for runtime
bytecode and artifact input, `solidity run` for one compiled deploy/call, and
confirmed Ethereum Mainnet transaction replay backed by exact RPC prestate.
Cancun remains EchoEVM's only fully declared ruleset; replay preserves the
actual transaction fork and explicit compatibility warnings.
