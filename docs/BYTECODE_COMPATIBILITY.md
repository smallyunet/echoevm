# Bytecode compatibility contract

EchoEVM declares transaction and legacy-bytecode execution for Cancun, Prague,
and Osaka. The executor is embedded `revm`; EchoEVM owns the fork selection,
transaction and block environment, state database, result normalization, trace,
replay, and native/Wasm integration around that engine.

## Executable evidence

The machine-readable regression matrix is
[`tests/conformance/bytecode-vectors.json`](../tests/conformance/bytecode-vectors.json).
It freezes 15 purpose-built vectors across 11 required categories and all three
declared forks. Native tests compare exact status, return or revert data, gas,
halt class, error, and normalized opcode sequence. The Chrome Wasm build runs
the same vectors and compares status, data, gas, and error.

The matrix also freezes the 154 opcode bytes registered by the pinned
`revm 42.0.1` dependency. Registration is not an activation claim: for example,
`CLZ` is registered but rejected before Osaka, while EOF-only `DUPN` and the
future `SLOTNUM` instruction remain rejected in declared legacy-bytecode rules.

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
- EOF-only instructions are registered by the dependency but are not active in
  the declared legacy-bytecode rulesets.
- Native and Wasm use different crypto backends where required; the shared
  vector gate proves current covered behavior, not universal backend equality.
- Updating `revm`, the official fixture release, a fork mapping, or a crypto
  feature requires reviewing and deliberately updating the frozen counts.
