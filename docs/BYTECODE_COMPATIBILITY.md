# Bytecode compatibility contract

EchoEVM declares transaction and legacy-bytecode execution for Cancun, Prague,
and Osaka. EchoEVM implements its own opcode interpreter, gas accounting,
transaction validation, call/create frames, journaled state, fork activation,
and precompiles. Native and Wasm frontends invoke that same Rust implementation;
no external EVM engine participates in execution or fixture comparison.

## Executable evidence

The machine-readable regression matrix is
[`tests/conformance/bytecode-vectors.json`](../tests/conformance/bytecode-vectors.json).
It freezes 15 purpose-built vectors across 11 required categories and all three
declared forks. Native tests compare exact status, return or revert data, gas,
halt class, error, and normalized opcode sequence. The Chrome Wasm build runs
the same vectors and compares status, data, gas, and error.

The matrix also freezes EchoEVM's 170-name opcode inventory and its SHA-256
fingerprint. Inventory membership is not an activation claim: for example,
`CLZ` is rejected before Osaka, while EOF-only `DUPN` and the future `SLOTNUM`
instruction remain rejected in declared legacy-bytecode rules.

Run the focused gate with:

```bash
make test-bytecode-conformance
```

## Official fixtures

`make test-official-fixtures` executes the pinned `tests@v20.0.1` state-test
corpora under their matching fork rules and requires exact, zero-skip totals:

| Fork | Files | Transactions | Accepted | Rejected |
|---|---:|---:|---:|---:|
| Cancun | 63 | 1,456 | 1,303 | 153 |
| Prague | 134 | 2,195 | 1,998 | 197 |
| Osaka | 187 | 3,461 | 3,244 | 217 |

The official state-test gate compares acceptance, account inventory, balance,
nonce, code, and storage. The regression vectors add exact gas, output, halt,
and trace checks. Neither gate claims full block validation or every fixture
family in the downloaded archive.

## Explicit boundaries

- Pre-Cancun replay is outside the declared v1 scope.
- Full block transition, Prague requests/system calls, and consensus validation
  are not bytecode/interpreter compatibility claims.
- EOF-only instructions may appear in EchoEVM's metadata inventory but are not
  active in the declared legacy-bytecode rulesets.
- The shared native/Wasm vector gate proves current covered behavior, not every
  possible host or embedding environment.
- Updating EchoEVM opcode semantics, the official fixture release, a fork
  mapping, or a crypto implementation requires reviewing and deliberately
  updating the frozen counts.
