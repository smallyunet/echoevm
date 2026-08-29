# Transaction witness replay

Use standalone replay for a self-contained `echoevm.replay-witness.v1` file.
Use the pinned official fixtures when the task asks for conformance evidence.

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

For a deterministic verdict/root-cause envelope, route the same witness through
the explanation facade:

```bash
echoevm explain replay ./transaction.witness.json \
  --expect-status success --format json
```

This command retains the same no-RPC replay boundary. Do not replace a missing
`rootCause` or `insufficient-evidence` verdict with an unsupported inference.

## Optional witness import

For migration or fixture acquisition, a configured trace-capable RPC may import exact prestate into a witness:

```bash
echoevm witness import-debug <transaction-hash-or-etherscan-url> \
  --out transaction.witness.json
```

Keep credentials in `ETHEREUM_RPC_URL` or a user-supplied `--rpc-url` and
never print credential-bearing URLs. The importer is a data-acquisition adapter;
the generated witness must replay later without RPC access.

For any transaction position in a block, prefer proof-verified acquisition when the provider supports the required standard RPC methods. Later positions replay the preceding transaction prefix locally from proved parent state:

```bash
echoevm witness import-proof <transaction-hash-or-etherscan-url> \
  --out transaction.witness.json \
  --proofs-out transaction.proofs.json
```

This path verifies EIP-1186 account and storage proofs against the parent state root and fails closed for later transactions. Standard RPC does not expose their intermediate prestate.

## Interpretation

- Verify witness chain, block, transaction, and fork provenance.
- EchoEVM selects Cancun, Prague, or Osaka transaction/interpreter semantics
  from the Mainnet block timestamp.
- Preserve compatibility warnings for pre-Cancun execution and absent
  historical BLOCKHASH entries.
- Report the pinned fixture release and exact case evidence for conformance work.
- Never describe a debug-RPC reference as part of EchoEVM standalone execution.
