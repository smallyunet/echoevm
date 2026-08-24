# Test witness contract

`echoevm.test-witness.v1` is a strict, self-contained input for one call-level
test. EchoEVM executes the runtime bytecode itself and does not invoke Foundry,
contact RPC, or consume another engine's execution result. Witnesses without a
`context` retain the original isolated empty-state call behavior. Witnesses with
a `context` run through EchoEVM's transaction/state path.

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
- `context.caller` and `context.target`: distinct transaction accounts.
- `context.value` and `context.gasPrice`: optional transaction values.
- `context.accounts`: explicit balance, nonce, code, and storage prestate. The
  caller must be present. `bytecode` is installed at the target; conflicting
  `accounts[target].code` is rejected.
- `context.environment`: chain ID, block number/time, coinbase, block gas limit,
  base fee, randomness, blob base fee, and historical block hashes.

Unknown fields are rejected. Account and storage reads not declared by a
context-bearing witness fail as incomplete rather than being guessed as zero. A
non-empty `requires` list fails with `unsupported-capability`; current examples
include `foundry-cheatcodes`, `foundry-set-up`, `rpc-fork`, and
`multi-transaction`.

```bash
echoevm explain test ./failure.test-witness.json --format json
```

## Foundry artifact exporter

The built-in exporter ABI-encodes one function call from a linked Foundry JSON
artifact. State must be supplied explicitly; repeat `--storage` for every slot
the execution may read. Caller, target, value, chain/block environment, base
fee, and randomness have explicit flags; inspect `from-foundry --help` for the
complete bounded input surface.

```bash
echoevm witness from-foundry out/StateReader.sol/StateReader.json \
  --function 'read()' \
  --storage 0x0000000000000000000000000000000000000000000000000000000000000000=0x000000000000000000000000000000000000000000000000000000000000002a \
  --expect-return 0x000000000000000000000000000000000000000000000000000000000000002a \
  --out failure.test-witness.json
echoevm explain test failure.test-witness.json --format json
```

This is an artifact-to-call exporter, not a replacement for Forge's test
harness. An ABI-visible `setUp()` or embedded standard HEVM cheatcode address is
recorded in `requires`, so the exported witness is inspectable but rejected by
execution. RPC forks, multi-transaction histories, dynamically reached
cheatcodes, and constructor effects must be materialized by a producer before
they can be executed. Forge failure text or Forge traces are never treated as
EchoEVM execution evidence.
