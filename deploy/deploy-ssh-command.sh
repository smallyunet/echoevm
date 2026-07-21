#!/usr/bin/env bash
set -euo pipefail

read -r action digest extra <<< "${SSH_ORIGINAL_COMMAND:-}"
if [[ "$action" != "deploy" || -n "${extra:-}" || ! "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
  echo "this SSH key may only deploy an EchoEVM image digest" >&2
  exit 2
fi

exec /usr/local/sbin/deploy-echoevm "$digest"
