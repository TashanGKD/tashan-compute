#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
script="$repo_root/deploy/install-aup-prerequisites.sh"
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT
log="$test_root/ssh.log"
fake_ssh="$test_root/ssh"
printf '%s\n' '#!/bin/sh' 'printf "%s\n" "$*" >>"$TCOMPUTE_TEST_LOG"' 'case "$*" in *hostname*) echo aup-test-01 ;; esac' >"$fake_ssh"
chmod +x "$fake_ssh"

usage=$(TCOMPUTE_DEPLOY_TESTING=1 TCOMPUTE_SSH_BIN="$fake_ssh" TCOMPUTE_TEST_LOG="$log" bash "$script")
grep -Fq 'Usage:' <<<"$usage"
[ ! -e "$log" ]

if TCOMPUTE_DEPLOY_TESTING=1 TCOMPUTE_SSH_BIN="$fake_ssh" TCOMPUTE_TEST_LOG="$log" bash "$script" --apply --host wrong-host >/dev/null 2>&1; then
  echo 'accepted wrong host' >&2
  exit 1
fi

TCOMPUTE_DEPLOY_TESTING=1 TCOMPUTE_SSH_BIN="$fake_ssh" TCOMPUTE_TEST_LOG="$log" bash "$script" --apply --host aup-server >/dev/null
grep -Fq 'apt-get install -y incus=6.0.0-1ubuntu0.3 incus-client uidmap=1:4.13+dfsg1-4ubuntu3 qemu-system-x86' "$log"

echo 'install-aup-prerequisites self-test: PASS'
