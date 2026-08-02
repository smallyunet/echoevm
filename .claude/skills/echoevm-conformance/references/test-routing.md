# Test routing

Select focused checks from the changed semantic surface, then run the complete conformance gate.

| Changed area | Focused checks |
|---|---|
| `internal/evm/core` | `go test ./internal/evm/core` |
| `internal/evm/vm/op_*` or `instructions_*` | Matching VM test file, then `go test ./internal/evm/vm` |
| CALL, STATICCALL, DELEGATECALL | `go test ./internal/evm/vm -run 'Call|Static'` and differential tests |
| CREATE, CREATE2, rollback | `go test ./internal/evm/vm -run 'Create|Rollback'` and differential tests |
| Gas accounting | Affected VM tests plus `make test-differential` |
| Precompiles | Matching precompile tests plus `make test-differential` |
| State, journaling, trie | State/core/trie unit tests plus integration tests |
| Replay | `go test ./internal/replay ./internal/web` plus replay service tests |
| Differential normalization | `go test ./internal/differential ./tests/differential` |
| CLI behavior | `go test ./cmd/echoevm ./tests/e2e` |

Use `-count=1` when checking a suspected cache-sensitive failure. Use `-race` for concurrency or shared-state changes.
