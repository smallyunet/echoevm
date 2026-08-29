#!/usr/bin/env bash
set -euo pipefail
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
set +e
output="$(ECHOEVM_OFFICIAL_FIXTURES="$repo_dir/tests/official/fixtures" cargo test --locked -p echoevm-core --features official-fixtures --test official --test official_block --test official_transaction -- --nocapture 2>&1)"
qstatus=$?
set -e
printf '%s\n' "$output"
if [[ "$qstatus" -ne 0 ]]; then
  exit "$qstatus"
fi
grep -Fq 'OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files=2337 transactions=11554 accepted=10968 rejected=586 fork=Cancun skipped=0' <<<"$output"
grep -Fq 'OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files=2471 transactions=13851 accepted=13063 rejected=788 fork=Prague skipped=0' <<<"$output"
grep -Fq 'OFFICIAL EXECUTION SUMMARY release=tests@v20.0.1 files=2408 transactions=14516 accepted=13708 rejected=808 fork=Osaka skipped=0' <<<"$output"
grep -Fq 'OFFICIAL BLOCK SUMMARY release=tests@v20.0.1 blocks=3 forks=3 skipped=0' <<<"$output"
grep -Fq 'OFFICIAL BLOCK CORPUS SUMMARY release=tests@v20.0.1 files=2401 accepted_single_blocks=11930 declared_rejected=748 fork=Cancun skipped=0' <<<"$output"
grep -Fq 'OFFICIAL BLOCK CORPUS SUMMARY release=tests@v20.0.1 files=2573 accepted_single_blocks=14621 declared_rejected=1286 fork=Prague skipped=0' <<<"$output"
grep -Fq 'OFFICIAL BLOCK CORPUS SUMMARY release=tests@v20.0.1 files=2514 accepted_single_blocks=15371 declared_rejected=1286 fork=Osaka skipped=0' <<<"$output"
grep -Fq 'OFFICIAL TRANSACTION SUMMARY release=tests@v20.0.1 files=1 valid=0 declared_rejected=1 fork=Cancun skipped=0' <<<"$output"
grep -Fq 'OFFICIAL TRANSACTION SUMMARY release=tests@v20.0.1 files=13 valid=0 declared_rejected=56 fork=Prague skipped=0' <<<"$output"
grep -Fq 'OFFICIAL TRANSACTION SUMMARY release=tests@v20.0.1 files=13 valid=0 declared_rejected=56 fork=Osaka skipped=0' <<<"$output"
