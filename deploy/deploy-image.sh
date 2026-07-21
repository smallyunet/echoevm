#!/usr/bin/env bash
set -euo pipefail

digest="${1:-}"
deploy_root=/opt/echoevm
env_file="$deploy_root/.env"
compose_file="$deploy_root/docker-compose.yml"
image="ghcr.io/smallyunet/echoevm@$digest"

if [[ ! "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
  echo "invalid image digest" >&2
  exit 2
fi
if [[ ! -f "$compose_file" ]]; then
  echo "compose file not found" >&2
  exit 2
fi

previous_image="$(sed -n 's/^ECHOEVM_IMAGE=//p' "$env_file" 2>/dev/null || true)"
printf 'ECHOEVM_IMAGE=%s\n' "$image" > "$env_file.next"
chmod 0644 "$env_file.next"
mv "$env_file.next" "$env_file"

cd "$deploy_root"
if docker compose pull echoevm && docker compose up -d --remove-orphans --wait --wait-timeout 45 echoevm; then
  echo "deployed $image"
  exit 0
fi

echo "deployment failed; rolling back" >&2
if [[ -n "$previous_image" ]]; then
  printf 'ECHOEVM_IMAGE=%s\n' "$previous_image" > "$env_file.next"
  chmod 0644 "$env_file.next"
  mv "$env_file.next" "$env_file"
  docker compose pull echoevm
  docker compose up -d --remove-orphans --wait --wait-timeout 45 echoevm
else
  docker compose down
fi
exit 1
