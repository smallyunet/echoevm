#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"
output="$(cargo test --locked -p echoevm-core --test bytecode_conformance -- --nocapture 2>&1)"
printf '%s\n' "$output"
grep -Fq 'BYTECODE CONFORMANCE SUMMARY vectors=15 categories=11 forks=3 registered_opcodes=154 skipped=0' <<<"$output"
