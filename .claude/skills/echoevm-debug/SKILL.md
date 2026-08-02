---
name: echoevm-debug
description: Compile, inspect, execute, trace, and differentially compare Solidity contracts, EVM bytecode, or confirmed Ethereum Mainnet transactions with EchoEVM and embedded Geth. Use when debugging a Solidity function, validating AI-generated contract changes, investigating a revert or gas discrepancy, replaying a transaction hash or Etherscan URL, or requesting opcode-level execution evidence.
---

# EchoEVM Debug

Use EchoEVM as the deterministic evidence engine. Explain the result only after collecting execution evidence.

## Resolve capabilities

1. Resolve `<skill-dir>` as the directory containing this `SKILL.md`.
2. Prefer a connected EchoEVM MCP server when it exposes the required logical operation.
3. Otherwise use the local `echoevm` CLI with JSON output.
4. Run `echoevm version --json` before the first CLI operation.
5. For Solidity input, also verify the selected compiler with `solc --version` or the configured compiler equivalent.
6. If neither MCP nor CLI is available, report the missing capability and stop. Do not invent execution results.

## Route the request

- For a `.sol` file, contract, ABI function, constructor, revert, or generated Solidity change, read [references/local-execution.md](references/local-execution.md).
- For bytecode, calldata, storage, opcode, or isolated gas behavior, read [references/local-execution.md](references/local-execution.md).
- For a transaction hash or Etherscan transaction URL, read [references/replay.md](references/replay.md).
- Before interpreting any result, read [references/evidence.md](references/evidence.md).

## Keep execution bounded

- Use only paths within the user's authorized workspace.
- Do not send local Solidity source, compiler inputs, or workspace files to a hosted service.
- Treat replay as read-only, but note that it calls the configured trace-capable Ethereum RPC.
- Write large JSON results to a temporary file. Run `python3 <skill-dir>/scripts/compact_result.py <result.json>` before loading the result into model context.
- Read a wider trace window only when the compact result does not contain enough evidence.

## Report the result

Return these sections in order:

1. Verdict.
2. Execution evidence: input, fork, engine versions, status, gas, state/storage, and trace match.
3. First relevant divergence or failure location.
4. Interpretation, explicitly labeled as inference when it goes beyond the evidence.
5. Limitations.
6. Suggested next check.

Never describe one matching input as complete EVM compatibility or a security audit.
