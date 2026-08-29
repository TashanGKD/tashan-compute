#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
gate="$repo_root/deploy/check-coder-stack.sh"

if [[ ! -x "$gate" ]]; then
  echo "missing executable gate: deploy/check-coder-stack.sh" >&2
  exit 1
fi

fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/deploy"
cp "$repo_root/deploy/versions.env" "$fixture/deploy/versions.env"
cp "$repo_root/deploy/compose.coder.yml" "$fixture/deploy/compose.coder.yml"
cp "$repo_root/deploy/coder.env.example" "$fixture/deploy/coder.env.example"
cp "$repo_root/deploy/tcompute-coder.service" "$fixture/deploy/tcompute-coder.service"

"$gate" --root "$fixture" >/dev/null

expect_rejection() {
  local name=$1
  local expected=$2
  shift 2
  cp "$repo_root/deploy/versions.env" "$fixture/deploy/versions.env"
  cp "$repo_root/deploy/compose.coder.yml" "$fixture/deploy/compose.coder.yml"
  cp "$repo_root/deploy/coder.env.example" "$fixture/deploy/coder.env.example"
  cp "$repo_root/deploy/tcompute-coder.service" "$fixture/deploy/tcompute-coder.service"
  "$@"
  if output=$("$gate" --root "$fixture" 2>&1); then
    echo "gate accepted $name" >&2
    exit 1
  fi
  grep -Fq "$expected" <<<"$output" || {
    echo "wrong rejection for $name: $output" >&2
    exit 1
  }
}

expect_rejection \
  "floating PostgreSQL tag" \
  "PostgreSQL image must include the pinned digest" \
  sed -i.bak 's/@sha256:[a-f0-9]\{64\}//' "$fixture/deploy/compose.coder.yml"

expect_rejection \
  "public PostgreSQL bind" \
  "PostgreSQL must bind only to 127.0.0.1" \
  sed -i.bak 's/127\.0\.0\.1:55432/0.0.0.0:55432/' "$fixture/deploy/compose.coder.yml"

expect_rejection \
  "socket exposure" \
  "workspace control plane must not mount Docker or Incus sockets" \
  sh -c 'printf "      - /var/run/docker.sock:/var/run/docker.sock\n" >> "$1/deploy/compose.coder.yml"' _ "$fixture"

expect_rejection \
  "committed password" \
  "deployment sources must not contain password assignments" \
  sh -c 'printf "POSTGRES_PASSWORD=not-a-real-password\n" >> "$1/deploy/coder.env.example"' _ "$fixture"

grep -Fq 'TF_PLUGIN_CACHE_DIR=/var/lib/tcompute/terraform-plugin-cache' "$repo_root/deploy/tcompute-coder.service" || {
  echo 'Coder service does not persist the Terraform provider cache' >&2
  exit 1
}

echo "check-coder-stack self-test: PASS"
