# EVM Conformance

EchoEVM uses two complementary, offline conformance layers. Neither layer is
allowed to silently skip cases.

## Pinned official fixtures

`tests/compliance/fixtures/` contains adapted cases from a fixed commit of the
official Ethereum legacy execution tests. Every fixture records its repository,
commit, source file, fork, and behavior category. The runner verifies return
data, halt status, execution errors, balances, nonces, code, and selected
storage slots.

The current baseline contains 9 Cancun ADD, MUL, and SUB cases. CI rejects empty
fixture files, missing provenance, missing case categories, or a baseline below
9 executed cases.

## Geth differential vectors

`tests/differential/` runs the same Cancun bytecode through EchoEVM and
go-ethereum v1.15.11. It compares:

- return or revert data;
- gas used;
- halt class: success, REVERT, or exceptional fault;
- persistent storage slot zero.

The current baseline contains 17 vectors across 8 categories: arithmetic,
bitwise, control, crypto, environment, fault, memory, and storage. CI requires
at least 15 vectors and at least one vector in every required category.

## Run the report

```bash
make test-conformance
```

Successful output includes two machine-searchable lines:

```text
COMPLIANCE SUMMARY official=9 categories=arithmetic/add=3,... forks=Cancun=9 skipped=0
DIFFERENTIAL SUMMARY fork=Cancun total=17 categories=... skipped=0
```

These numbers are a transparent regression baseline, not a claim of complete
Ethereum consensus compatibility. Full GeneralStateTests coverage remains a
long-term roadmap item.
