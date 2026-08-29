#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
script="$repo_root/deploy/configure-incus.sh"
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT
log="$test_root/ssh.log"
fake_ssh="$test_root/ssh"
cat >"$fake_ssh" <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >>"$TCOMPUTE_TEST_LOG"
case "$*" in
  *hostname*) echo aup-test-01 ;;
esac
EOF
chmod +x "$fake_ssh"

usage=$(TCOMPUTE_SSH_BIN="$fake_ssh" TCOMPUTE_TEST_LOG="$log" bash "$script")
grep -Fq 'Usage:' <<<"$usage"
[[ ! -e "$log" ]]

for bad_host in wrong-host 'aup-server;touch-pwned'; do
  if TCOMPUTE_SSH_BIN="$fake_ssh" TCOMPUTE_TEST_LOG="$log" \
    bash "$script" --apply --host "$bad_host" >/dev/null 2>&1; then
    echo "accepted invalid host: $bad_host" >&2
    exit 1
  fi
done

TCOMPUTE_SSH_BIN="$fake_ssh" TCOMPUTE_TEST_LOG="$log" \
  bash "$script" --apply --host aup-server >/dev/null

for required in \
  'restricted.containers.nesting=allow' \
  'restricted.devices.disk=allow' \
  'restricted.devices.disk.paths' \
  'restricted.devices.gpu=block' \
  'security.acls=tcompute-egress' \
  'security.acls.default.ingress.action=allow' \
  'security.acls.default.egress.action=allow' \
  '169.254.0.0/16' \
  '10.0.0.0/8' \
  'fc00::/7' \
  'limits.cpu=30' \
  'limits.memory=48GiB'; do
  grep -Fq "$required" "$log" || {
    echo "missing confinement command: $required" >&2
    exit 1
  }
done

if grep -Fq '240.0.0.0/4' "$repo_root/deploy/configure-incus.sh"; then
  echo 'ACL blocks the DHCP broadcast range' >&2
  exit 1
fi

echo 'configure-incus self-test: PASS'
