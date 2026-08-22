#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
extension_dir="$(cd "${script_dir}/.." && pwd)"
repo_dir="$(cd "${extension_dir}/../.." && pwd)"
manifest_version="$(node -p "require('${extension_dir}/manifest.json').version")"
release_tag="${ECHOEVM_RELEASE_TAG:-}"

if [[ -n "${release_tag}" && "v${manifest_version}" != "${release_tag}" ]]; then
  echo "Chrome extension version v${manifest_version} does not match release tag ${release_tag}" >&2
  exit 1
fi

staging_dir="${repo_dir}/build/chrome-extension"
fixture_path="${repo_dir}/build/chrome-wasm-test-witness.json"
asset_path="${repo_dir}/dist/echoevm-chrome-${manifest_version}.zip"
rm -rf "${staging_dir}"
mkdir -p "${staging_dir}/wasm" "${staging_dir}/icons" "${repo_dir}/dist"

cp "${extension_dir}/manifest.json" "${extension_dir}/background.js" "${extension_dir}/lib.js" "${extension_dir}/content.js" "${extension_dir}/content.css" "${staging_dir}/"
cp "${extension_dir}/popup.html" "${extension_dir}/popup.js" "${extension_dir}/popup.css" "${staging_dir}/"
cp "${extension_dir}/THIRD_PARTY_NOTICES.md" "${repo_dir}/LICENSE" "${staging_dir}/"
cp "${extension_dir}"/icons/icon-*.png "${staging_dir}/icons/"
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "${staging_dir}/wasm/"

GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${staging_dir}/wasm/engine.wasm" ./cmd/echoevm-wasm

node "${extension_dir}/scripts/validate.mjs" "${staging_dir}"
go run "${extension_dir}/test/generate_fixture.go" "${fixture_path}"
node "${extension_dir}/scripts/smoke-wasm.mjs" "${staging_dir}" "${fixture_path}"

rm -f "${asset_path}"
(
  cd "${staging_dir}"
  zip -q -r "${asset_path}" .
)

echo "Built ${asset_path}"
echo "Load unpacked: ${staging_dir}"
