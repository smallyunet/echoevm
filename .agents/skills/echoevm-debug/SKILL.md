---
name: echoevm-debug
description: Explain EVM-sensitive Solidity, bytecode, or self-contained Ethereum transaction witnesses with bounded EchoEVM execution evidence. Use when debugging reverts, gas, storage, low-level calls, contract creation, bytecode, or transaction replay; skip routine source edits that ordinary tests already explain.
---

# EchoEVM Debug

Use EchoEVM as a local deterministic execution microscope. Base conclusions on its structured output; use the conformance workflow, not a one-off trace, for claims about EchoEVM correctness or compatibility.

## Resolve capabilities

1. Use the local `echoevm` CLI and record `echoevm version --json` before the first execution.
2. For Solidity input, also record the selected compiler version.
3. If the CLI or required compiler is unavailable, report the missing capability. Do not invent execution results or silently substitute another EVM.

## Route the request

- For an EVM-sensitive `.sol` file, contract, ABI function, constructor, revert, gas boundary, storage change, or low-level call, read [references/local-execution.md](references/local-execution.md).
- For bytecode, calldata, storage, opcode, or isolated gas behavior, read [references/local-execution.md](references/local-execution.md).
- For an existing witness, transaction hash, or Etherscan transaction URL, read [references/replay.md](references/replay.md). A hash or URL requires an explicit acquisition step before replay.
- Before interpreting any result, read [references/evidence.md](references/evidence.md).

## Keep execution bounded

- Do not send local Solidity source, compiler inputs, or workspace files to a hosted service.
- Treat standalone witness replay as local and read-only. It must not contact an RPC. `witness import-debug` is an optional fixture-development acquisition adapter, not an execution backend.
- Route bounded evidence with `auto`, `revert`, `storage`, `call`, `abi`, `arithmetic`, or `gas`. `--limit` changes presentation after complete execution; it does not limit execution.
- If bounded evidence is insufficient, rerun the identical input with `--format json` and inspect only the relevant portion of the result. The current CLI does not provide step-window or opcode/depth filter flags.
- Start with one representative input. Add at most one branch-distinct input before a mismatch; use the project's existing tests for broad input coverage.

## Report the result

Report the execution cause in this order:

1. Verdict.
2. Execution evidence: input, status, gas, state/storage, and selected trace scope.
3. First relevant step, call depth, PC, opcode, gas, and stack transition present in the output.
4. Interpretation, explicitly labeled as inference when it goes beyond the evidence.
5. Limitations.
6. Suggested next check.

Never describe one execution as complete EVM compatibility, a security audit, or formal verification.
