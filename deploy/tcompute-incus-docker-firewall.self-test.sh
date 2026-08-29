#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
script="$repo_root/deploy/tcompute-incus-docker-firewall.sh"
unit="$repo_root/deploy/tcompute-incus-docker-firewall.service"
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT
log="$test_root/commands.log"

cat >"$test_root/ip" <<'EOF'
#!/bin/sh
printf 'ip %s\n' "$*" >>"$TCOMPUTE_TEST_LOG"
exit 0
EOF
cat >"$test_root/iptables" <<'EOF'
#!/bin/sh
printf 'iptables %s\n' "$*" >>"$TCOMPUTE_TEST_LOG"
case "$*" in
  '-C DOCKER-USER -i incusbr-955 -j ACCEPT')
    grep -Fq -- '-I DOCKER-USER 1 -i incusbr-955 -j ACCEPT' "$TCOMPUTE_TEST_LOG"; exit $? ;;
  '-C DOCKER-USER -o incusbr-955 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT')
    grep -Fq -- '-I DOCKER-USER 1 -o incusbr-955 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT' "$TCOMPUTE_TEST_LOG"; exit $? ;;
esac
exit 0
EOF
cat >"$test_root/id" <<'EOF'
#!/bin/sh
test "$1" = -u
test "$2" = tcompute
echo 955
EOF
chmod +x "$test_root/ip" "$test_root/iptables" "$test_root/id"

usage=$(TCOMPUTE_IP_BIN="$test_root/ip" TCOMPUTE_IPTABLES_BIN="$test_root/iptables" \
  TCOMPUTE_ID_BIN="$test_root/id" TCOMPUTE_TEST_LOG="$log" bash "$script")
grep -Fq 'Usage:' <<<"$usage"
[[ ! -e "$log" ]]

TCOMPUTE_IP_BIN="$test_root/ip" TCOMPUTE_IPTABLES_BIN="$test_root/iptables" \
  TCOMPUTE_ID_BIN="$test_root/id" TCOMPUTE_TEST_LOG="$log" bash "$script" --apply >/dev/null
grep -Fq 'iptables -I DOCKER-USER 1 -i incusbr-955 -j ACCEPT' "$log"
grep -Fq 'iptables -I DOCKER-USER 1 -o incusbr-955 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT' "$log"

grep -Fq 'ExecStart=/usr/local/lib/tashan-compute/tcompute-incus-docker-firewall.sh --apply' "$unit"
grep -Fq 'After=docker.service incus.service' "$unit"
grep -Fq 'PartOf=docker.service' "$unit"

echo 'tcompute-incus-docker-firewall self-test: PASS'
