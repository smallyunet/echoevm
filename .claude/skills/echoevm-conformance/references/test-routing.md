# Test routing

Select focused checks from the changed semantic surface, then run the complete conformance gate.

| Changed area | Focused checks |
|---|---|
| Opcode dispatch or instruction semantics under `crates/echoevm-core/src/engine/` | Closest `echoevm-core` test, then `make test-bytecode-conformance` |
| CALL, STATICCALL, DELEGATECALL, CREATE, CREATE2, or rollback | `cargo test -p echoevm-core --locked`, then `make test-bytecode-conformance` |
| Gas accounting or fork activation | `cargo test -p echoevm-core --locked`, bytecode conformance, then the relevant official fork |
| Precompiles (`bls.rs`, `bn254.rs`, `kzg.rs`, or `engine/precompiles.rs`) | Matching Rust unit tests, bytecode conformance, then the relevant official fork |
| State, transaction, authorization, or trie root | `cargo test -p echoevm-core --locked`, then the relevant official fork |
| Replay or witness execution | Core replay tests plus `bash scripts/test-cli.sh`; test-witness changes also run `make test-test-witness`; acquisition-only changes need their focused CLI tests |
| Protocol or evidence selection | Affected crate tests plus CLI and native/Wasm bytecode compatibility checks |
| CLI or Solidity facade | `bash scripts/test-cli.sh`; Solidity protocol changes also run `make test-integration` |
| Wasm execution surface | `make test-bytecode-conformance` and `make test-chrome` |

Use `ECHOEVM_OFFICIAL_FORK=Cancun`, `Prague`, or `Osaka` only for focused diagnosis. The complete release claim requires `make test-conformance-full` without a fork filter and with zero skipped cases.
