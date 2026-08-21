# Mainnet transaction replay

Use replay only for a confirmed Ethereum Mainnet transaction hash or an allowlisted Etherscan transaction URL.

## Preconditions

- Require an Ethereum Mainnet RPC endpoint with `debug_traceTransaction`, `debug_traceCall`, `prestateTracer`, and opcode tracing.
- Prefer an EchoEVM MCP `replay_transaction` operation when available.
- Otherwise use the CLI and keep RPC credentials in `ECHOEVM_ETHEREUM_RPC` or the configured environment. Never print the RPC URL when it contains credentials.

## CLI workflow

```bash
echoevm replay <transaction-hash-or-etherscan-url> \
  --format evidence-json \
  --profile auto \
  --limit 40
```

Route a known failure to `--profile revert`, state questions to `storage`, call
structure to `call`, and gas questions to `gas`. Check transaction/fork
provenance, warnings, comparison confidence, and selection/truncation metadata
before interpreting the selected events.

Only when compact evidence is insufficient, request the full replay and compact
it before loading the result into model context:

```bash
echoevm replay <transaction-hash-or-etherscan-url> --format json > <temporary-result.json>
python3 <skill-dir>/scripts/compact_result.py <temporary-result.json>
```

Use `--rpc-url` only when the user explicitly supplies a safe endpoint and the command output will not expose it.

## Interpretation

- Verify chain ID 1 and confirmed block metadata.
- Report the transaction's actual fork label.
- EchoEVM executes Cancun semantics. For transactions from other fork eras, preserve the compatibility warning and do not call the replay exact.
- Distinguish transaction status, return data, gas, post-state, and trace mismatches.
- Treat missing transaction, pending transaction, upstream RPC failure, missing tracer support, and timeout as different failure classes.
- Never approximate prestate from the parent block when the required tracer is unavailable.
