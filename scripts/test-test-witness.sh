#!/usr/bin/env bash
set -euo pipefail
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
binary="$repo_dir/target/debug/echoevm"
fixtures="$repo_dir/benchmarks/test-explain/fixtures"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

"$binary" explain test "$fixtures/arithmetic-return-mismatch.json" --format json | jq -e '
  .schema == "echoevm.explanation.v1" and
  .input.kind == "test-witness" and
  .verdict.code == "expectation-mismatch" and
  .rootCause.code == "arithmetic-input-provenance" and
  any(.rootCause.evidence[]; .pc == 4 and .op == "DIV" and .source.file == "Arithmetic.t.sol")
'

"$binary" explain test "$fixtures/storage-mismatch.json" --format json | jq -e '
  .verdict.code == "expectation-mismatch" and
  .rootCause.code == "storage-write" and
  any(.rootCause.evidence[]; .pc == 4 and .op == "SSTORE" and .source.file == "Storage.t.sol")
'

"$binary" explain test "$fixtures/expected-revert-but-success.json" --format json | jq -e '
  .verdict.code == "insufficient-evidence" and .rootCause == null
'

"$binary" explain test "$fixtures/unexpected-revert.json" --format json | jq -e '
  .verdict.code == "expectation-mismatch" and
  .rootCause.code == "execution-revert" and
  any(.rootCause.evidence[]; .op == "REVERT")
'

if "$binary" explain test "$fixtures/unsupported-cheatcodes.json" --format json \
  >"$scratch/unsupported.out" 2>"$scratch/unsupported.err"; then
  echo "unsupported capability witness unexpectedly succeeded" >&2
  exit 1
fi
grep -Fq 'unsupported-capability: foundry-cheatcodes' "$scratch/unsupported.err"

slot="0x0000000000000000000000000000000000000000000000000000000000000000"
value="0x000000000000000000000000000000000000000000000000000000000000002a"
"$binary" witness from-foundry "$repo_dir/benchmarks/test-explain/foundry/StateReader.json" \
  --function 'read()' --storage "$slot=$value" --expect-status success \
  --expect-return "$value" --out "$scratch/stateful.json"
jq -e '
  .schema == "echoevm.test-witness.v1" and
  .context.accounts["0x2000000000000000000000000000000000000002"].storage != null and
  .source.function == "read()"
' "$scratch/stateful.json"
"$binary" explain test "$scratch/stateful.json" --format json | jq -e '
  .verdict.code == "execution-completed" and
  .execution.returnData == "0x000000000000000000000000000000000000000000000000000000000000002a"
'

"$binary" witness from-foundry "$repo_dir/benchmarks/test-explain/foundry/StateReader.json" \
  --function 'read()' --expect-status success --out "$scratch/incomplete.json"
if "$binary" explain test "$scratch/incomplete.json" --format json \
  >"$scratch/incomplete.out" 2>"$scratch/incomplete.err"; then
  echo "undeclared storage read unexpectedly succeeded" >&2
  exit 1
fi
grep -Fq 'execution witness is incomplete' "$scratch/incomplete.err"

"$binary" witness from-foundry "$repo_dir/benchmarks/test-explain/foundry/SetUpTest.json" \
  --function 'testExample()' --out "$scratch/setup.json"
jq -e '.requires == ["foundry-set-up"]' "$scratch/setup.json"
if "$binary" explain test "$scratch/setup.json" --format json \
  >"$scratch/setup.out" 2>"$scratch/setup.err"; then
  echo "setUp-dependent witness unexpectedly succeeded" >&2
  exit 1
fi
grep -Fq 'unsupported-capability: foundry-set-up' "$scratch/setup.err"
echo "Test witness explanation gate passed"
