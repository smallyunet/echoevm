# EchoEVM conformance contract

EchoEVM v1.8.0 calls a release **A-grade** only when every gate below passes
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

The same pinned release adds accepted single-block transition coverage:

| Fork | Files | Accepted single blocks | Declared rejected inventory |
|---|---:|---:|---:|
| Cancun | 2,401 | 11,930 | 748 |
| Prague | 2,573 | 14,621 | 1,286 |
| Osaka | 2,514 | 15,371 | 1,286 |
| **Total** | **7,488** | **41,922** | **3,320** |

Each accepted single-block case verifies signed transaction decoding, protocol
system calls, withdrawals, and exact header commitments for transaction root,
withdrawals root, gas used, receipts root, logs bloom, and final state root. The
3,320 rejected cases are a pinned inventory, not an executed rejection claim.

The transaction fixture gate also pins 113 declared-invalid cases: one Cancun,
56 Prague, and 56 Osaka. It checks malformed/trailing encodings and invalid
EIP-7702 authorization-list formats and signatures. This fixture release does
not contain an accepted transaction case in those selected directories.

## Independent gates

The release additionally requires:

- the 15-vector native/Wasm bytecode matrix across 11 semantic categories;
- focused Rust unit and integration tests;
- CLI, Solidity, replay-witness, stateful test-witness, Foundry preparation,
  Chrome/Wasm, VS Code, and packaged-skill tests;
- formatting, lint, locked dependency, and release artifact checks in CI.

Run the complete local conformance gate with:

```bash
make test-conformance-full
```

`ECHOEVM_OFFICIAL_FORK=Cancun`, `Prague`, or `Osaka` may be used for focused
diagnosis. Release evidence always runs all three without that filter.

## Boundaries

A-grade covers EchoEVM's declared Cancun-through-Osaka transaction surface,
legacy bytecode state transitions, and the accepted single-block fixture gate
above. It does not cover pre-Cancun rules, rejected or multi-block blockchain
fixture execution, consensus-layer validation, independent `requestsHash`
recomputation, Engine API/Hive interoperability, networking, database
durability, sync, transaction-pool policy, or every official fixture family.

Passing this contract supports the scoped statement “EchoEVM passes the pinned
official state-test and accepted single-block gates for its declared forks.” It
does not support “the Ethereum Foundation certified EchoEVM” or “EchoEVM is a
drop-in replacement for a production execution client.”
