# Block witness contract

`echoevm.block-witness.v1` is the complete offline input for sequential block
execution. The executor never contacts RPC or delegates semantics to another EVM.

## Fields

- `schema`: exactly `echoevm.block-witness.v1`.
- `chainId`: execution chain ID for protected transaction signatures.
- `fork`: explicit `Cancun`, `Prague`, or `Osaka` ruleset.
- `blockHash`: canonical hash of `header`.
- `header`: full Ethereum execution header.
- `transactions`: ordered EIP-2718 signed transaction bytes.
- `withdrawals`: ordered execution-layer withdrawals when declared by the header.
- `prestate`: complete parent-state accounts, code, and storage touched by the
  block, including protocol system contracts. `exists: false` distinguishes a
  proven absent account from an allocated account whose code is empty;
  `storageComplete: true` declares all omitted slots known-zero.
- `blockHashes`: optional preceding block hashes observable through `BLOCKHASH`.
- `source`: optional descriptive acquisition provenance.

Unknown fields, an invalid schema, empty prestate, empty transaction bytes, and
inputs larger than 64 MiB are rejected.

## Execution

```bash
echoevm block ./block.witness.json
echoevm block ./block.witness.json --trace-transaction 3
```

Transactions execute in order against one shared world state. EchoEVM applies
beacon-root processing on Cancun and later, Prague/Osaka history and request
system calls, and withdrawals. It then verifies the header hash, transaction
root, withdrawals root, cumulative gas used, receipts root, logs bloom, and
final state root. Missing account or storage reads fail closed.

The optional zero-based `--trace-transaction` keeps the other transaction
results compact while attaching an opcode trace to the selected result.

## Boundary

Version 1 accepts one already selected execution block and an explicit fork. It
does not validate consensus-layer rules, fork choice, Engine API payload status,
network propagation, or chain ancestry. Prague/Osaka request system calls are
executed, but the emitted request list is not yet retained to independently
recompute `requestsHash`; the result carries an explicit warning when relevant.
