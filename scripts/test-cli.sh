#!/usr/bin/env bash
set -euo pipefail
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
binary="$repo_dir/target/debug/echoevm"
cargo build --locked -p echoevm
"$binary" version --json | jq -e '.version == "1.5.1"'
"$binary" run 60016002015f5260205ff3 --json | jq -e '.status == "success" and .returnData == "0x0000000000000000000000000000000000000000000000000000000000000003"'
[[ "$("$binary" trace 600160020100 | wc -l | tr -d ' ')" == "4" ]]
"$binary" disasm 602a00 | grep -Fq 'PUSH1 0x2a'
"$binary" deploy 60006000f3 --json | jq -e '.status == "success"'
witness="$(mktemp)"
trap 'rm -f "$witness"' EXIT
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
echo "CLI protocol smoke passed"
