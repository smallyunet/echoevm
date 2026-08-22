---
name: echoevm-debug
description: Explain EVM-sensitive Solidity, bytecode, or confirmed Ethereum Mainnet execution with bounded, customizable EchoEVM opcode evidence. Use when debugging reverts, gas boundaries, storage, low-level calls, CREATE/CREATE2, bytecode, transaction replay, or opcode-level behavior; skip routine high-level Solidity edits that normal tests fully cover.
---

# EchoEVM Debug

Use EchoEVM as the deterministic execution microscope. Prefer its explainable
opcode process over any external comparison; use official fixtures for conformance
about compatibility or EchoEVM correctness.

## Resolve capabilities

1. Resolve `<skill-dir>` as the directory containing this `SKILL.md`.
2. Prefer a connected EchoEVM MCP server when it exposes the explainable trace operation.
3. Otherwise use the local `echoevm` CLI. Start with bounded `echoevm.evidence.v1` using `--format evidence-json --profile auto --limit 40` for opcode questions and `summary-json` for execution-result questions. Solidity source runs and standalone witness replay support the same evidence format.
4. Run `echoevm version --json` before the first CLI operation.
5. For Solidity input, also verify the selected compiler with `solc --version` or the configured compiler equivalent.
6. If neither MCP nor CLI is available, report the missing capability and stop. Do not invent execution results.

## Route the request

- For an EVM-sensitive `.sol` file, contract, ABI function, constructor, revert, gas boundary, storage change, or low-level call, read [references/local-execution.md](references/local-execution.md).
- For bytecode, calldata, storage, opcode, or isolated gas behavior, read [references/local-execution.md](references/local-execution.md).
- For a transaction hash or Etherscan transaction URL, read [references/replay.md](references/replay.md).
- Before interpreting any result, read [references/evidence.md](references/evidence.md).

## Keep execution bounded

- Use only paths within the user's authorized workspace.
- Do not send local Solidity source, compiler inputs, or workspace files to a hosted service.
- Treat standalone witness replay as local and read-only. It must not contact an RPC. `witness import-debug` is an optional fixture-development acquisition adapter, not an execution backend.
- Route evidence with `auto`, `revert`, `storage`, `call`, `abi`, `arithmetic`, or `gas`; use full trace fields, opcode/depth filters, and windows only when compact evidence is insufficient.
- If the trace result is truncated, request a deterministic `--around-step` window; do not load the whole trace by default.
- Request standalone replay evidence directly from the witness. Compact large evidence JSON before loading it into model context.
- Start with one representative input. Add at most one branch-distinct input before a mismatch; use the project's existing tests for broad input coverage.

## Report the result

Report the execution cause in this order:

1. Verdict.
2. Execution evidence: input, status, gas, state/storage, and selected trace scope.
3. First relevant step, call depth, PC, opcode, and structured state delta.
4. Interpretation, explicitly labeled as inference when it goes beyond the evidence.
5. Limitations.
6. Suggested next check.

When a comparison was explicitly required, also report engine versions, match
fields, and first divergence. Never describe one execution as complete EVM
compatibility or a security audit.
