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

`internal/differential/` provides the production EchoEVM and embedded Geth
runners used by the CLI, local Web Explorer, and `tests/differential/`. The
test suite runs the same Cancun bytecode through go-ethereum v1.17.4 and
compares:

- return or revert data;
- gas used;
- halt class: success, REVERT, or exceptional fault;
- observed persistent storage slots;
- normalized top-level opcode sequence, gas, and reliably aligned stack state;
- the first field and step where the normalized executions diverge.

The normalized trace treats PC, opcode, gas, and stack captured immediately
before an opcode as the shared anchor. Post-op gas and non-terminal stack are
derived from the next top-level pre-op callback. Terminal stack and memory are
not compared because the two tracer APIs do not expose them with equivalent
semantics.

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
