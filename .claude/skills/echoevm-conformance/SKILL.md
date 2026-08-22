---
name: echoevm-conformance
description: Validate EchoEVM interpreter, opcode, gas, state, call-frame, and fork-semantics changes against focused tests, pinned official Ethereum fixtures, and independent regression vectors. Use when modifying internal EVM execution code, diagnosing conformance regressions, reviewing execution-semantics changes, or determining the test impact of an opcode or gas-accounting change.
---

# EchoEVM Conformance

Validate execution-semantics changes with the narrowest reliable evidence first, then expand to the complete conformance gate.

## Inspect scope

1. Read the changed files and relevant tests before running commands.
2. Preserve unrelated worktree changes.
3. Read [references/test-routing.md](references/test-routing.md) and select focused packages and vectors.
4. Treat Cancun through Osaka as declared transaction/interpreter rulesets; keep block-level and pre-Cancun boundaries explicit.

## Validate progressively

1. Run the closest unit tests for the affected opcode, state component, call frame, or precompile.
2. Run the relevant integration or independent regression package.
3. Run `make test-conformance` for any execution-semantics change.
4. Run `make test` before declaring a release-ready result.
5. In restricted environments, place `GOPATH`, `GOCACHE`, and `GOMODCACHE` under a task-specific directory in `/tmp`.

## Protect the baseline

Read [references/conformance-contract.md](references/conformance-contract.md) before interpreting the suite. Require:

- Exact official-fixture and regression-vector totals from the test output.
- Zero skipped execution.
- Required category coverage and non-shrinking minimum guards.
- Comparison of status, return or revert data, gas, halt class, storage/state, and normalized trace where applicable.

Do not update an expected result merely to make a failing test pass. Establish whether an official fixture, a protocol specification, or a documented unsupported fork boundary is the relevant oracle.

## Report the result

State the changed semantic surface, focused checks, full conformance counts, skips, first failure or divergence, and remaining uncertainty. A passing curated baseline is release evidence, not a claim of complete Ethereum compatibility.
