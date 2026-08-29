#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: check-coder-stack.sh [--root PATH]

Read-only gate for the Coder control-plane deployment sources.
EOF
}

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
while (($#)); do
  case "$1" in
    --root)
      [[ $# -ge 2 && -n "$2" ]] || { echo "--root requires a path" >&2; exit 2; }
      root=$2
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

versions="$root/deploy/versions.env"
compose="$root/deploy/compose.coder.yml"
env_example="$root/deploy/coder.env.example"
unit="$root/deploy/tcompute-coder.service"
for required in "$versions" "$compose" "$env_example" "$unit"; do
  [[ -f "$required" ]] || { echo "required deployment source missing: $required" >&2; exit 1; }
done

read_version() {
  local key=$1
  local value
  value=$(awk -F= -v key="$key" '$1 == key {sub(/^[^=]*=/, ""); print; found=1} END {if (!found) exit 1}' "$versions") || {
    echo "missing version pin: $key" >&2
    exit 1
  }
  [[ -n "$value" ]] || { echo "empty version pin: $key" >&2; exit 1; }
  printf '%s' "$value"
}

postgres_image=$(read_version POSTGRES_IMAGE)
postgres_digest=$(read_version POSTGRES_IMAGE_DIGEST)
expected_image="$postgres_image@$postgres_digest"

grep -Fq "image: $expected_image" "$compose" || {
  echo "PostgreSQL image must include the pinned digest: $expected_image" >&2
  exit 1
}

grep -Fq '"127.0.0.1:55432:5432"' "$compose" || {
  echo "PostgreSQL must bind only to 127.0.0.1" >&2
  exit 1
}

if grep -Eiq '(docker\.sock|incus/(unix\.socket|socket)|/var/lib/incus)' "$compose" "$unit"; then
  echo "workspace control plane must not mount Docker or Incus sockets" >&2
  exit 1
fi

if grep -Eiq '(^|[[:space:]])[A-Z0-9_]*PASSWORD[[:space:]]*=[[:space:]]*[^[:space:]#]+' \
  "$versions" "$compose" "$env_example" "$unit"; then
  echo "deployment sources must not contain password assignments" >&2
  exit 1
fi

grep -Fq 'CODER_HTTP_ADDRESS=127.0.0.1:7080' "$env_example" || {
  echo "Coder HTTP must bind only to 127.0.0.1" >&2
  exit 1
}

grep -Fq 'User=tcompute' "$unit" || {
  echo "Coder service must run as tcompute" >&2
  exit 1
}

if grep -Eq '^User=(root|aup)$' "$unit"; then
  echo "Coder service must not run as root or aup" >&2
  exit 1
fi

echo "check-coder-stack: PASS"
