# EchoEVM conformance contract

EchoEVM v1.5.0 calls a release **A-grade** only when every gate below passes
without skipped or expected-failure cases. This is a project release label, not
an Ethereum Foundation certification and not a claim of full execution-client
equivalence.

## Pinned official gate

The oracle is the Ethereum execution-spec fixture release `tests@v20.0.1`.
EchoEVM consumes every state-test file in the directories matching its declared
forks:

| Fork | Files | Transactions | Accepted | Rejected |
|---|---:|---:|---:|---:|
| Cancun | 2,337 | 11,554 | 10,968 | 586 |
| Prague | 2,471 | 13,851 | 13,063 | 788 |
| Osaka | 2,408 | 14,516 | 13,708 | 808 |
| **Total** | **7,216** | **39,921** | **37,739** | **2,182** |

For each accepted transaction the runner requires:

- canonical EIP-2718 signed bytes after decode/re-encode and recovered sender;
- exact receipt status and cumulative gas used;
- exact ordered EVM logs via their RLP logs hash;
- exact account inventory, nonce, balance, code, and non-zero storage;
- exact post-state Merkle-Patricia root.

For each rejected transaction it requires a matching normalized official
`TransactionException` category, unchanged expected state root, and the expected
empty logs commitment. The only compatibility alias is an older Prague ported
fixture that labels an EIP-7623 floor-gas failure with the broader
`INTRINSIC_GAS_TOO_LOW` category.

The runner also pins every file, accepted, rejected, and total transaction
count. Corpus shrinkage, a missing fork directory, or a skipped case fails the
gate.

## Independent gates

The release additionally requires:

- the 15-vector native/Wasm bytecode matrix across 11 semantic categories;
- focused Rust unit and integration tests;
- CLI, Solidity, replay-witness, Chrome/Wasm, VS Code, and packaged-skill tests;
- formatting, lint, locked dependency, and release artifact checks in CI.

Run the complete local conformance gate with:

```bash
make test-conformance-full
```

`ECHOEVM_OFFICIAL_FORK=Cancun`, `Prague`, or `Osaka` may be used for focused
diagnosis. Release evidence always runs all three without that filter.

## Boundaries

A-grade covers EchoEVM's declared Cancun-through-Osaka transaction and legacy
bytecode state-transition surface. It does not cover pre-Cancun rules, block
assembly/import, withdrawals or Prague request processing, consensus-layer
validation, Engine API/Hive interoperability, networking, database durability,
sync, transaction-pool policy, or every non-state-test fixture family.

Passing this contract supports the scoped statement “EchoEVM passes the pinned
official state-test corpus for its declared forks.” It does not support “the
Ethereum Foundation certified EchoEVM” or “EchoEVM is a drop-in replacement for
a production execution client.”
