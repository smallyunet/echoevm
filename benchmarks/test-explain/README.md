# Self-contained test explanation fixtures

These fixtures exercise `echoevm explain test` with strict
`echoevm.test-witness.v1` inputs. Supported witnesses contain runtime bytecode,
calldata, gas/fork metadata, expectations, and optional source locations. They
execute locally with an empty initial state.

The protocol fails closed when `requires` is non-empty. Current unsupported
capabilities include Foundry cheatcodes, RPC forks, setup-derived initial state,
and multi-transaction test sequences. A future Foundry exporter must either
materialize those requirements into a supported self-contained witness or
preserve them in `requires`; it must never silently drop them.

Run `make test-test-witness` to exercise all supported and rejected fixtures.
The gate also exports a stateful call from the checked-in Foundry artifact and
verifies both successful replay and incomplete-storage rejection.
