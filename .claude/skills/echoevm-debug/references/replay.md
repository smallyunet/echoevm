# Transaction witness replay

Use standalone replay for a self-contained `echoevm.replay-witness.v1` file.
Use online verification only when the task explicitly asks to compare EchoEVM
with a Geth-compatible RPC for conformance.

## Standalone replay

Replay must not require an RPC URL or another execution engine:

```bash
echoevm replay ./transaction.witness.json \
  --format evidence-json \
  --profile auto \
  --limit 40
```

Check the witness schema and SHA-256 provenance, transaction/fork metadata,
warnings, execution result, and selection/truncation metadata before
interpreting events. Treat an incomplete or invalid witness as an input failure;
do not fetch missing state implicitly or invent it.

Only when compact evidence is insufficient, request the full standalone result:

```bash
echoevm replay ./transaction.witness.json --format json
```

## Optional witness import

For migration or conformance, a configured trace-capable RPC may import exact
prestate into a witness:

```bash
echoevm witness import-debug <transaction-hash-or-etherscan-url> \
  --out transaction.witness.json
```

Keep credentials in `ECHOEVM_ETHEREUM_RPC` or a user-supplied `--rpc-url` and
never print credential-bearing URLs. The importer is a data-acquisition adapter;
the generated witness must replay later without RPC access.

## Optional engine verification

Use this only when comparison was requested:

```bash
echoevm verify <transaction-hash-or-etherscan-url> \
  --format evidence-json \
  --profile auto \
  --limit 40
```

Verification requires Mainnet `debug_traceTransaction`, `debug_traceCall`,
`prestateTracer`, and opcode tracing. Distinguish transaction status, return
data, gas, post-state, and trace mismatches. Treat missing transaction, pending
transaction, upstream RPC failure, missing tracer support, and timeout as
different failure classes.

## Interpretation

- Verify the witness or online transaction chain, block, and fork provenance.
- EchoEVM selects Cancun, Prague, or Osaka transaction/interpreter semantics
  from the Mainnet block timestamp.
- Preserve compatibility warnings for pre-Cancun execution and absent
  historical BLOCKHASH entries.
- Report Geth version, match fields, and first divergence only for `verify`.
- Never describe a debug-RPC reference as part of EchoEVM standalone execution.
