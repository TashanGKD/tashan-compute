#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
script="$repo_root/deploy/verify-incus-isolation.sh"
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT
log="$test_root/ssh.log"
fake_ssh="$test_root/ssh"
cat >"$fake_ssh" <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >>"$TCOMPUTE_TEST_LOG"
case "$*" in
  *hostname*) echo aup-test-01 ;;
  *tcompute-isolation-smoke*) echo isolation-smoke=PASS ;;
esac
EOF
chmod +x "$fake_ssh"

usage=$(TCOMPUTE_SSH_BIN="$fake_ssh" TCOMPUTE_TEST_LOG="$log" bash "$script")
grep -Fq 'Usage:' <<<"$usage"
[[ ! -e "$log" ]]

if TCOMPUTE_SSH_BIN="$fake_ssh" TCOMPUTE_TEST_LOG="$log" \
  bash "$script" --smoke --host 'aup-server;touch-pwned' >/dev/null 2>&1; then
  echo 'accepted injected host' >&2
  exit 1
fi

TCOMPUTE_SSH_BIN="$fake_ssh" TCOMPUTE_TEST_LOG="$log" \
  bash "$script" --smoke --host aup-server >/dev/null

for required in \
  'trap cleanup EXIT INT TERM' \
  'test ! -S /var/lib/incus/unix.socket' \
  '/dev/tcp/example.com/443' \
  '/dev/tcp/169.254.169.254/80' \
  '/dev/tcp/100.126.44.113/22' \
  'test "$host_uid" != 0' \
  'security.idmap.isolated:'; do
  grep -Fq "$required" "$log" || {
    echo "missing live isolation assertion: $required" >&2
    exit 1
  }
done

echo 'verify-incus-isolation self-test: PASS'
