#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
manifest="$repo_dir/tests/official/manifest.json"
cache="$repo_dir/tests/official/.cache/fixtures.tar.gz"
destination="$repo_dir/tests/official/fixtures"
expected="$(jq -r .sha256 "$manifest")"
url="$(jq -r .url "$manifest")"

if [[ -d "$destination/state_tests/for_osaka" ]]; then
  echo "official fixtures already installed: $destination"
  exit 0
fi
mkdir -p "$(dirname "$cache")"
if [[ ! -f "$cache" ]] || [[ "$(shasum -a 256 "$cache" | awk '{print $1}')" != "$expected" ]]; then
  curl --fail --location --retry 3 --output "$cache" "$url"
fi
actual="$(shasum -a 256 "$cache" | awk '{print $1}')"
[[ "$actual" == "$expected" ]] || { echo "fixture checksum mismatch" >&2; exit 1; }
temporary="$(mktemp -d "${TMPDIR:-/tmp}/echoevm-eest.XXXXXX")"
trap 'rm -rf "$temporary"' EXIT
tar -xzf "$cache" -C "$temporary"
root="$(find "$temporary" -type d -path '*/state_tests/for_osaka' -print -quit)"
[[ -n "$root" ]] || { echo "fixture archive has no state_tests/for_osaka" >&2; exit 1; }
fixture_root="${root%/state_tests/for_osaka}"
mv "$fixture_root" "$destination"
echo "installed and verified: $destination"
