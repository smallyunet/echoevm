#!/usr/bin/env bash
set -euo pipefail

digest="${1:-}"
deploy_root="${ECHOEVM_DEPLOY_ROOT:-/opt/echoevm}"
deploy_script_path="${ECHOEVM_DEPLOY_SCRIPT_PATH:-/usr/local/sbin/deploy-echoevm}"
env_file="$deploy_root/.env"
compose_file="$deploy_root/docker-compose.yml"
caddy_file="$deploy_root/Caddyfile"
image="ghcr.io/smallyunet/echoevm@$digest"
bundle_path=/usr/local/share/echoevm/deploy
container_id=""

umask 077

if [[ ! "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
  echo "invalid image digest" >&2
  exit 2
fi

mkdir -p "$deploy_root"
work_root="$(mktemp -d "$deploy_root/.deploy.XXXXXX")"
bundle_root="$work_root/bundle"
backup_root="$work_root/backup"
next_env="$work_root/env.next"
mkdir -p "$bundle_root" "$backup_root"

cleanup() {
	if [[ -n "$container_id" ]]; then
		docker rm -f "$container_id" >/dev/null 2>&1 || true
	fi
	rm -rf "$work_root"
}
trap cleanup EXIT

backup_file() {
	local source="$1" name="$2"
	if [[ -f "$source" ]]; then
		cp -p "$source" "$backup_root/$name"
		touch "$backup_root/$name.exists"
	fi
}

restore_file() {
	local target="$1" name="$2" mode="$3"
	if [[ -f "$backup_root/$name.exists" ]]; then
		install -m "$mode" "$backup_root/$name" "$target"
	else
		rm -f "$target"
	fi
}

write_next_env() {
	if [[ -f "$env_file" ]]; then
		awk -v image="$image" '
			BEGIN { replaced = 0 }
			/^ECHOEVM_IMAGE=/ {
				if (!replaced) print "ECHOEVM_IMAGE=" image
				replaced = 1
				next
			}
			{ print }
			END { if (!replaced) print "ECHOEVM_IMAGE=" image }
		' "$env_file" > "$next_env"
	else
		printf 'ECHOEVM_IMAGE=%s\n' "$image" > "$next_env"
	fi
	chmod 0600 "$next_env"
}

compose() {
	docker compose --env-file "$env_file" -f "$compose_file" "$@"
}

backup_file "$env_file" env
backup_file "$compose_file" compose
backup_file "$caddy_file" caddy
backup_file "$deploy_script_path" deploy-script

docker pull "$image" >/dev/null
container_id="$(docker create "$image")"
docker cp "$container_id:$bundle_path/." "$bundle_root"
docker rm "$container_id" >/dev/null
container_id=""

for required in docker-compose.yml Caddyfile deploy-image.sh; do
	if [[ ! -s "$bundle_root/$required" ]]; then
		echo "deployment bundle is missing $required" >&2
		exit 2
	fi
done

write_next_env
docker compose --env-file "$next_env" -f "$bundle_root/docker-compose.yml" config --quiet
docker run --rm --network none \
	-v "$bundle_root/Caddyfile:/etc/caddy/Caddyfile:ro" \
	caddy:2.11.4-alpine caddy validate --config /etc/caddy/Caddyfile >/dev/null

install -m 0600 "$next_env" "$env_file"
install -m 0644 "$bundle_root/docker-compose.yml" "$compose_file"
install -m 0644 "$bundle_root/Caddyfile" "$caddy_file"

cd "$deploy_root"
if compose pull && \
	compose up -d --remove-orphans --wait --wait-timeout 60 && \
	compose exec -T echoevm wget -q -O /dev/null http://127.0.0.1:8080/readyz; then
	install -m 0755 "$bundle_root/deploy-image.sh" "$deploy_script_path.next"
	mv "$deploy_script_path.next" "$deploy_script_path"
	echo "deployed $image"
	exit 0
fi

echo "deployment failed; rolling back" >&2
restore_file "$env_file" env 0600
restore_file "$compose_file" compose 0644
restore_file "$caddy_file" caddy 0644
restore_file "$deploy_script_path" deploy-script 0755
if [[ -f "$compose_file" && -f "$env_file" ]]; then
	compose pull
	compose up -d --remove-orphans --wait --wait-timeout 60
else
	docker compose --env-file "$next_env" -f "$bundle_root/docker-compose.yml" down || true
fi
exit 1
