---
name: echoevm-debug
description: Explain EVM-sensitive Solidity, bytecode, or confirmed Ethereum Mainnet execution with bounded, customizable EchoEVM opcode evidence. Use when debugging reverts, gas boundaries, storage, low-level calls, CREATE/CREATE2, bytecode, transaction replay, or opcode-level behavior; skip routine high-level Solidity edits that normal tests fully cover.
---

# EchoEVM Debug

Use EchoEVM as the deterministic execution microscope. Prefer its explainable
opcode process over a Geth comparison; compare engines only when the task asks
about compatibility or EchoEVM correctness.

## Resolve capabilities

1. Resolve `<skill-dir>` as the directory containing this `SKILL.md`.
2. Prefer a connected EchoEVM MCP server when it exposes the explainable trace operation.
3. Otherwise use the local `echoevm` CLI. Start with a bounded `echoevm.trace.v1` view for opcode questions and `summary-json` for execution-result questions.
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
- Treat replay as read-only, but note that it calls the configured trace-capable Ethereum RPC.
- Select fields, opcodes, depth, and changes-only output before loading trace data into model context.
- If the trace result is truncated, request a deterministic `--around-step` window; do not load the whole trace by default.
- Write legacy differential or replay JSON results to a temporary file. Run `python3 <skill-dir>/scripts/compact_result.py <result.json>` before loading those results into model context.
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
