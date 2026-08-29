#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  tcompute-incus-docker-firewall.sh --check
  tcompute-incus-docker-firewall.sh --apply

With no arguments this command only prints help. --apply installs two fixed,
idempotent Docker/Incus forwarding rules for the dedicated tcompute bridge.
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
if [[ "$mode" == help ]]; then
  usage
  exit 0
fi

ip_bin=${TCOMPUTE_IP_BIN:-ip}
iptables_bin=${TCOMPUTE_IPTABLES_BIN:-iptables}
id_bin=${TCOMPUTE_ID_BIN:-id}
uid=$($id_bin -u tcompute)
[[ "$uid" =~ ^[0-9]+$ ]] || { echo 'invalid tcompute uid' >&2; exit 1; }
bridge="incusbr-$uid"

$ip_bin link show "$bridge" >/dev/null
$iptables_bin -S DOCKER-USER >/dev/null

outbound=(-i "$bridge" -j ACCEPT)
returning=(-o "$bridge" -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT)

if [[ "$mode" == apply ]]; then
  $iptables_bin -C DOCKER-USER "${outbound[@]}" 2>/dev/null || \
    $iptables_bin -I DOCKER-USER 1 "${outbound[@]}"
  $iptables_bin -C DOCKER-USER "${returning[@]}" 2>/dev/null || \
    $iptables_bin -I DOCKER-USER 1 "${returning[@]}"
fi

$iptables_bin -C DOCKER-USER "${outbound[@]}"
$iptables_bin -C DOCKER-USER "${returning[@]}"
echo "Docker/Incus forwarding verified for $bridge"
