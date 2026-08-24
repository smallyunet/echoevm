#!/usr/bin/env bash
set -euo pipefail
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output="$(ECHOEVM_OFFICIAL_FIXTURES="$repo_dir/tests/official/fixtures" cargo test --locked -p echoevm-core --features official-fixtures --test official -- --nocapture 2>&1)"
printf '%s\n' "$output"
grep -Fq 'OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files=2337 transactions=11554 accepted=10968 rejected=586 fork=Cancun skipped=0' <<<"$output"
grep -Fq 'OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files=2471 transactions=13851 accepted=13063 rejected=788 fork=Prague skipped=0' <<<"$output"
grep -Fq 'OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files=2408 transactions=14516 accepted=13708 rejected=808 fork=Osaka skipped=0' <<<"$output"
