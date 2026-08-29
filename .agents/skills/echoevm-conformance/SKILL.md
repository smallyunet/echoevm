---
name: echoevm-conformance
description: Validate changes to EchoEVM's Rust execution semantics against focused tests, independent bytecode vectors, and pinned official Ethereum fixtures. Use when modifying or reviewing opcode, gas, transaction, state, call-frame, precompile, fork, or replay execution code; do not invoke for documentation-only or frontend-only changes.
---

# EchoEVM Conformance

Validate Rust execution-semantics changes with focused evidence first, then expand to the gate required by the claim being made.

## Inspect scope

1. Inspect the changed semantic surface and relevant tests.
2. Read [references/test-routing.md](references/test-routing.md) and select the closest Rust and public-interface checks.
3. Treat Cancun through Osaka as the declared transaction/interpreter scope and accepted single-block fixture gate; keep pre-Cancun, rejected/multi-block, consensus, and networking behavior outside the claim.

## Validate progressively

1. Run the closest Rust unit or integration test.
2. Run the independent bytecode matrix when opcode, gas, state, call/create, fork, or precompile behavior can change.
3. Run `make test-conformance` before declaring an execution-semantics change locally validated.
4. Run `make test-conformance-full` before making the complete pinned-fixture or release conformance claim. This downloads and executes the full official corpus.
5. Run the additional release checks required by `docs/CONFORMANCE.md` before calling a release ready; `make test` alone is not the full conformance gate.

## Protect the baseline

Read [references/conformance-contract.md](references/conformance-contract.md) before interpreting the suite. Require:

- Exact official-fixture and regression-vector totals from the test output.
- Zero skipped execution.
- Required category coverage and non-shrinking minimum guards.
- Exact accept/reject category, receipt status and gas, logs, post-state, and state root where the official fixture provides them.

Do not update an expected result merely to make a failing test pass. Establish whether an official fixture, a protocol specification, or a documented unsupported fork boundary is the relevant oracle.

## Report the result

State the changed semantic surface, commands run, exact fixture/vector totals, skips, first failing case, and remaining uncertainty. Distinguish focused validation from the complete official-fixture gate. A passing project gate is scoped release evidence, not external certification or complete execution-client compatibility.
