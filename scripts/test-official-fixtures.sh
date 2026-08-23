#!/usr/bin/env bash
set -euo pipefail
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output="$(ECHOEVM_OFFICIAL_FIXTURES="$repo_dir/tests/official/fixtures" cargo test --locked -p echoevm-core --features official-fixtures --test official -- --nocapture 2>&1)"
printf '%s\n' "$output"
grep -Fq 'OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files=63 transactions=1456 accepted=1303 rejected=153 fork=Cancun skipped=0' <<<"$output"
grep -Fq 'OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files=134 transactions=2195 accepted=1998 rejected=197 fork=Prague skipped=0' <<<"$output"
grep -Fq 'OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files=187 transactions=3461 accepted=3244 rejected=217 fork=Osaka skipped=0' <<<"$output"
