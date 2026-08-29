#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  tcompute-ecs-tunnel-firewall.sh --check
  tcompute-ecs-tunnel-firewall.sh --apply

With no arguments this command only prints help. The fixed rules prevent direct
Internet access to the reverse-tunnel port while preserving ECS loopback access.
EOF
}

mode=help
if (($#)); then
  case "$1" in
    --check|--apply) mode=${1#--} ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
  shift
fi
[[ $# -eq 0 ]] || { echo 'unexpected extra arguments' >&2; exit 2; }
if [[ "$mode" == help ]]; then usage; exit 0; fi

iptables_bin=${TCOMPUTE_IPTABLES_BIN:-iptables}
ip6tables_bin=${TCOMPUTE_IP6TABLES_BIN:-ip6tables}
rule=(! -i lo -p tcp --dport 13980 -j DROP)

if [[ "$mode" == apply ]]; then
  $iptables_bin -C INPUT "${rule[@]}" 2>/dev/null || $iptables_bin -I INPUT 1 "${rule[@]}"
  $ip6tables_bin -C INPUT "${rule[@]}" 2>/dev/null || $ip6tables_bin -I INPUT 1 "${rule[@]}"
fi

$iptables_bin -C INPUT "${rule[@]}"
$ip6tables_bin -C INPUT "${rule[@]}"
echo 'ECS tunnel port 13980 is restricted to loopback'
