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
echo "Test witness explanation gate passed"
