#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo 'Usage: check-public-network.sh [--root PATH]'
}

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
while (($#)); do
  case "$1" in
    --root)
      [[ $# -ge 2 && -n "$2" ]] || { echo '--root requires a path' >&2; exit 2; }
      root=$2
      shift 2
      ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

nginx="$root/deploy/nginx/compute.tashan.chat.conf"
tunnel="$root/deploy/tcompute-coder-tunnel.service"
coder_env="$root/deploy/coder.env.example"
for required in "$nginx" "$tunnel" "$coder_env"; do
  [[ -f "$required" ]] || { echo "required public-network source missing: $required" >&2; exit 1; }
done

grep -Fq 'server_name compute.tashan.chat *.workspaces.compute.tashan.chat;' "$nginx" || {
  echo 'nginx must cover the dashboard and workspace wildcard hosts' >&2
  exit 1
}
[[ $(grep -Ec '^[[:space:]]*proxy_pass ' "$nginx") -eq 1 ]] && \
  grep -Fq 'proxy_pass http://127.0.0.1:13980;' "$nginx" || {
    echo 'nginx upstream must be ECS loopback port 13980' >&2
    exit 1
  }
grep -Fq 'proxy_set_header Upgrade $http_upgrade;' "$nginx" || {
  echo 'nginx must forward WebSocket upgrades' >&2
  exit 1
}

grep -Fq -- '-R 127.0.0.1:13980:127.0.0.1:7080' "$tunnel" || {
  if grep -Fq -- '-R 0.0.0.0:13980:' "$tunnel"; then
    echo 'tunnel must bind only to ECS loopback' >&2
  else
    echo 'tunnel must target Coder loopback port 7080' >&2
  fi
  exit 1
}
grep -Fq 'User=aup' "$tunnel" || { echo 'tunnel must run as aup' >&2; exit 1; }
if grep -Fq 'StrictHostKeyChecking=no' "$tunnel"; then
  echo 'tunnel must verify the ECS host key' >&2
  exit 1
fi

grep -Fq 'CODER_PROXY_TRUSTED_HEADERS=X-Forwarded-For' "$coder_env" || {
  echo 'Coder must explicitly trust the forwarded client-IP header' >&2
  exit 1
}
grep -Fq 'CODER_PROXY_TRUSTED_ORIGINS=127.0.0.1/32' "$coder_env" || {
  echo 'Coder must trust forwarded headers only from the local tunnel' >&2
  exit 1
}

echo 'check-public-network: PASS'
