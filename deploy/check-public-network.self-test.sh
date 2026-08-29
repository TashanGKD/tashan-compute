#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
gate="$repo_root/deploy/check-public-network.sh"
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/deploy/nginx"

for source_file in \
  deploy/nginx/compute.tashan.chat.conf \
  deploy/tcompute-coder-tunnel.service \
  deploy/coder.env.example; do
  mkdir -p "$fixture/$(dirname "$source_file")"
  cp "$repo_root/$source_file" "$fixture/$source_file"
done

"$gate" --root "$fixture" >/dev/null

sed -i.bak 's/127\.0\.0\.1:13980/0.0.0.0:13980/' "$fixture/deploy/tcompute-coder-tunnel.service"
if output=$("$gate" --root "$fixture" 2>&1); then
  echo 'gate accepted public reverse-forward bind' >&2
  exit 1
fi
grep -Fq 'tunnel must bind only to ECS loopback' <<<"$output"

cp "$repo_root/deploy/tcompute-coder-tunnel.service" "$fixture/deploy/tcompute-coder-tunnel.service"
sed -i.bak 's/127\.0\.0\.1:7080/127.0.0.1:8080/' "$fixture/deploy/tcompute-coder-tunnel.service"
if output=$("$gate" --root "$fixture" 2>&1); then
  echo 'gate accepted wrong AUP upstream' >&2
  exit 1
fi
grep -Fq 'tunnel must target Coder loopback port 7080' <<<"$output"

cp "$repo_root/deploy/tcompute-coder-tunnel.service" "$fixture/deploy/tcompute-coder-tunnel.service"
printf '\nproxy_pass http://10.0.0.5:13980;\n' >>"$fixture/deploy/nginx/compute.tashan.chat.conf"
if output=$("$gate" --root "$fixture" 2>&1); then
  echo 'gate accepted non-loopback nginx upstream' >&2
  exit 1
fi
grep -Fq 'nginx upstream must be ECS loopback port 13980' <<<"$output"

echo 'check-public-network self-test: PASS'
