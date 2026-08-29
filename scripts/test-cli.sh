#!/usr/bin/env bash
set -euo pipefail
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
binary="$repo_dir/target/debug/echoevm"
cargo build --locked -p echoevm
"$binary" version --json | jq -e '.version == "1.8.0"'
"$binary" run 60016002015f5260205ff3 --json | jq -e '.status == "success" and .returnData == "0x0000000000000000000000000000000000000000000000000000000000000003"'
[[ "$("$binary" trace 600160020100 | wc -l | tr -d ' ')" == "4" ]]
"$binary" disasm 602a00 | grep -Fq 'PUSH1 0x2a'
"$binary" behavior 600035631122334414600d57005b60043560015500 --format json | jq -e '
  .schema == "echoevm.behavior.v1" and
  .coverage.recognizedSelectors == 1 and
  .functions[0].effects[0].kind == "storage-write" and
  .functions[0].effects[0].inputs.value == "calldata.arg0"
'
"$binary" deploy 60006000f3 --json | jq -e '.status == "success"'
witness="$(mktemp)"
block_witness="$(mktemp)"
trap 'rm -f "$witness" "$block_witness"' EXIT
cargo run --locked -p echoevm-core --example generate-witness -- "$witness" >/dev/null
"$binary" explain replay "$witness" --format json | jq -e '
  .schema == "echoevm.explanation.v1" and
  .input.kind == "transaction-witness" and
  .verdict.code == "execution-completed" and
  .input.witness.sha256 != null
'
"$binary" explain replay "$witness" --format json --expect-return 0x01 | jq -e '
  .verdict.code == "insufficient-evidence" and .rootCause == null
'
printf '%s\n' '{"schema":"wrong"}' >"$block_witness"
if "$binary" block "$block_witness" >/dev/null 2>&1; then
  echo "block command accepted an invalid witness" >&2
  exit 1
fi
echo "CLI protocol smoke passed"
