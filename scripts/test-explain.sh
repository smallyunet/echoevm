#!/usr/bin/env bash
set -euo pipefail
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
generated="$(mktemp -d)"
trap 'rm -rf "$generated"' EXIT
python3 "$repo_dir/benchmarks/trace-value-v2/generate_fixtures.py" \
  --echoevm "$repo_dir/target/debug/echoevm" \
  --output "$generated"
diff -ru "$repo_dir/benchmarks/trace-value-v2/fixtures" "$generated"
echo "Explain fixture gate passed"
