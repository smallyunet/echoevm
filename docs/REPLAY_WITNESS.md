# Replay witness contract

`echoevm.replay-witness.v1` is the complete input contract for standalone
transaction replay. The replay executor reads this document, executes with
EchoEVM, and never contacts an RPC endpoint or another EVM engine.

## Fields

- `schema`: exactly `echoevm.replay-witness.v1`.
- `chainId`: execution chain ID; must match protected transaction signatures.
- `blockHash`: canonical hash of `header`.
- `transactionIndex`: transaction position in the source block.
- `header`: full Ethereum execution header used for block context.
- `transaction`: EIP-2718 binary transaction encoded as hex.
- `prestate`: address-keyed accounts containing balance, nonce, code, and every
  storage slot the transaction can read or write.
- `blockHashes`: optional decimal-block-number map for `BLOCKHASH`; only the
  preceding 256 blocks are observable by the EVM.
- `source`: optional acquisition provenance. It is descriptive and never
  changes execution.

The CLI rejects unknown top-level or account fields, malformed addresses,
empty state, mismatched header hashes, invalid transaction encoding, and chain
ID mismatch. Each result includes the witness schema and SHA-256 digest.

## Execution and acquisition boundaries

```bash
echoevm replay ./transaction.witness.json --format evidence-json
```

This command is the formal replay capability. It is deterministic for the
witness bytes and does not expose an RPC option.

`witness import-debug` is an explicitly named acquisition adapter for
capturing exact prestate from a provider that exposes `prestateTracer`:

```bash
echoevm witness import-debug 0x0123... \
  --rpc-url https://your-trace-rpc.example \
  --out transaction.witness.json
```

The imported file must replay later with no provider. The adapter's upstream
response is acquisition data, never the EchoEVM execution result or oracle.

## Completeness responsibility

Version 1 uses explicit prestate rather than implicit lazy network reads. A
witness producer must include every touched account and storage slot. EchoEVM
fails on structurally invalid witnesses, but a structurally valid witness that
omits an otherwise existing account or zero-valued slot cannot be proven
complete without Merkle proofs. A future proof-backed witness version will bind
all reads to a state root; until then, trusted acquisition or frozen reviewed
fixtures are required for historical Mainnet claims.
