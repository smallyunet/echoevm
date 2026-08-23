#!/usr/bin/env bash
set -euo pipefail
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
binary="$repo_dir/target/debug/echoevm"
cargo build --locked -p echoevm
"$binary" version --json | jq -e '.version == "1.3.0"'
"$binary" run 60016002015f5260205ff3 --json | jq -e '.status == "success" and .returnData == "0x0000000000000000000000000000000000000000000000000000000000000003"'
[[ "$("$binary" trace 600160020100 | wc -l | tr -d ' ')" == "4" ]]
"$binary" disasm 602a00 | grep -Fq 'PUSH1 0x2a'
"$binary" deploy 60006000f3 --json | jq -e '.status == "success"'
echo "CLI protocol smoke passed"
