#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
test_root="$(mktemp -d)"
trap 'rm -rf "$test_root"' EXIT
fake_bin="$test_root/bin"
mkdir -p "$fake_bin"

cat > "$fake_bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$FAKE_DOCKER_LOG"
case "${1:-}" in
	pull)
		exit 0
		;;
	create)
		printf '%s\n' fake-container
		;;
	cp)
		cp -R "$FAKE_BUNDLE_ROOT/." "$3"
		;;
	rm|run)
		exit 0
		;;
	compose)
		if [[ " $* " == *" exec -T echoevm "* && "${FAKE_READY_FAIL:-0}" == "1" ]]; then
			exit 1
		fi
		exit 0
		;;
	*)
		echo "unexpected docker command: $*" >&2
		exit 1
		;;
esac
EOF
chmod 0755 "$fake_bin/docker"

digest="sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

file_mode() {
	stat -f '%Lp' "$1" 2>/dev/null || stat -c '%a' "$1"
}

prepare_root() {
	local root="$1"
	mkdir -p "$root"
	cat > "$root/.env" <<'EOF'
ECHOEVM_ETHEREUM_RPC=https://rpc.example/private-token
ECHOEVM_IMAGE=ghcr.io/smallyunet/echoevm@sha256:old
CUSTOM_SETTING=preserve-me
EOF
	chmod 0644 "$root/.env"
	printf '%s\n' old-compose > "$root/docker-compose.yml"
	printf '%s\n' old-caddy > "$root/Caddyfile"
	printf '%s\n' '#!/usr/bin/env bash' 'echo old-deployer' > "$root/deploy-echoevm"
	chmod 0755 "$root/deploy-echoevm"
}

run_deploy() {
	local root="$1" log="$2"
	PATH="$fake_bin:$PATH" \
	FAKE_DOCKER_LOG="$log" \
	FAKE_BUNDLE_ROOT="$script_dir" \
	ECHOEVM_DEPLOY_ROOT="$root" \
	ECHOEVM_DEPLOY_SCRIPT_PATH="$root/deploy-echoevm" \
		bash "$script_dir/deploy-image.sh" "$digest"
}

success_root="$test_root/success"
success_log="$test_root/success.log"
prepare_root "$success_root"
run_deploy "$success_root" "$success_log"
grep -Fqx 'ECHOEVM_ETHEREUM_RPC=https://rpc.example/private-token' "$success_root/.env"
grep -Fqx 'CUSTOM_SETTING=preserve-me' "$success_root/.env"
grep -Fqx "ECHOEVM_IMAGE=ghcr.io/smallyunet/echoevm@$digest" "$success_root/.env"
[[ "$(grep -c '^ECHOEVM_IMAGE=' "$success_root/.env")" == "1" ]]
[[ "$(file_mode "$success_root/.env")" == "600" ]]
cmp "$script_dir/docker-compose.yml" "$success_root/docker-compose.yml"
cmp "$script_dir/Caddyfile" "$success_root/Caddyfile"
cmp "$script_dir/deploy-image.sh" "$success_root/deploy-echoevm"
grep -E 'compose .* up -d --remove-orphans --wait --wait-timeout 60$' "$success_log" >/dev/null
grep -E 'compose .* exec -T echoevm wget .*readyz$' "$success_log" >/dev/null

rollback_root="$test_root/rollback"
rollback_log="$test_root/rollback.log"
prepare_root "$rollback_root"
cp -p "$rollback_root/.env" "$test_root/expected.env"
cp -p "$rollback_root/docker-compose.yml" "$test_root/expected-compose.yml"
cp -p "$rollback_root/Caddyfile" "$test_root/expected-Caddyfile"
cp -p "$rollback_root/deploy-echoevm" "$test_root/expected-deployer"
if FAKE_READY_FAIL=1 run_deploy "$rollback_root" "$rollback_log"; then
	echo "deployment unexpectedly succeeded when readiness failed" >&2
	exit 1
fi
cmp "$test_root/expected.env" "$rollback_root/.env"
cmp "$test_root/expected-compose.yml" "$rollback_root/docker-compose.yml"
cmp "$test_root/expected-Caddyfile" "$rollback_root/Caddyfile"
cmp "$test_root/expected-deployer" "$rollback_root/deploy-echoevm"
[[ "$(file_mode "$rollback_root/.env")" == "600" ]]
[[ "$(grep -c 'compose .* up -d --remove-orphans --wait --wait-timeout 60$' "$rollback_log")" == "2" ]]

echo "deployment contract tests passed"
