# Test witness contract

`echoevm.test-witness.v1` is a strict, self-contained input for one call-level
test. EchoEVM executes the runtime bytecode from an empty initial state and does
not invoke Foundry, contact RPC, or consume another engine's execution result.
The execution address, caller, origin, call value, gas price, and block values
are zero/default values from the isolated runtime executor.

## Fields

- `schema`: exactly `echoevm.test-witness.v1`.
- `name`: non-empty test or reproduction name.
- `bytecode`: non-empty runtime bytecode.
- `calldata`: optional call data, default `0x`.
- `gasLimit`: non-zero execution gas limit.
- `fork`: `Cancun`, `Prague`, or `Osaka`; default `Osaka`.
- `expectation.status`: optional `success`, `revert`, or `fault`.
- `expectation.returnData`: optional exact bytes or a short zero-padded ABI word.
- `expectation.storage`: optional 32-byte slot/value assertions. Missing final
  slots are interpreted as zero.
- `source`: optional file, contract, function, test name, and exact top-level
  runtime PC source locations. Locations are never applied to nested frames.
- `requires`: capabilities required to reproduce the test but not materialized
  in the witness.

Unknown fields are rejected. A non-empty `requires` list fails with
`unsupported-capability`; current examples include `foundry-cheatcodes`,
`rpc-fork`, `initial-state`, and `multi-transaction`.

```bash
echoevm explain test ./failure.test-witness.json --format json
```

## Foundry exporter boundary

A Foundry adapter may export this protocol only when it can fully materialize a
single runtime call under the empty-state contract. If a test depends on
`setUp()`, `vm.*` cheatcodes, forked state, balances/accounts, constructor state,
or earlier transactions, the exporter must record those requirements and let
EchoEVM reject the witness. Parsing a Forge failure message or importing a Forge
trace as if EchoEVM executed it is outside this contract.

Supporting stateful Foundry tests requires a future execution-witness extension
with explicit accounts, storage, caller/value, and environment. Those fields
must not be silently added to or ignored by v1.
