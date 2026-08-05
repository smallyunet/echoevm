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

## Agent workflow

Start with a narrow changes-only view:

```bash
echoevm trace \
  --bin-runtime ./runtime.bin \
  --calldata 0x1234 \
  --changes-only \
  --fields gas,stack,storage,control,explanation \
  --limit 200 \
  --format json
```

If the result is truncated or a step needs more context, request a deterministic
window without rerunning a different input:

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

The explainable protocol currently fronts the standalone `trace` command for
runtime bytecode and artifact input. Solidity source-run and Mainnet replay
integration are subsequent milestones. Cancun remains EchoEVM's only fully
declared ruleset.
