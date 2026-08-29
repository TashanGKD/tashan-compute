#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
script="$repo_root/deploy/tcompute-ecs-tunnel-firewall.sh"
unit="$repo_root/deploy/tcompute-ecs-tunnel-firewall.service"

usage=$(bash "$script")
grep -Fq 'Usage:' <<<"$usage"
grep -Fq -- '! -i lo -p tcp --dport 13980 -j DROP' "$script"
grep -Fq 'ExecStart=/usr/local/lib/tashan-compute/tcompute-ecs-tunnel-firewall.sh --apply' "$unit"

echo 'tcompute-ecs-tunnel-firewall self-test: PASS'
