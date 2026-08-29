#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
versions="$repo_root/deploy/versions.env"

read_version() {
  local key=$1
  local value
  value=$(sed -n "s/^${key}=\([0-9A-Za-z.:+-]*\)$/\1/p" "$versions")
  [ -n "$value" ] || { echo "missing or invalid $key" >&2; exit 1; }
  printf '%s\n' "$value"
}

usage() {
  printf '%s\n' \
    'Usage:' \
    '  install-aup-prerequisites.sh --check --host aup-server' \
    '  install-aup-prerequisites.sh --apply --host aup-server' \
    '' \
    'With no arguments this command only prints help.'
}

mode=help
host=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --check | --apply)
      [ "$mode" = help ] || { echo 'choose exactly one mode' >&2; exit 1; }
      mode=${1#--}; shift ;;
    --host)
      [ "$#" -ge 2 ] || { echo '--host requires a value' >&2; exit 1; }
      host=$2; shift 2 ;;
    -h | --help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

if [ "$mode" = help ]; then usage; exit 0; fi
[ "$host" = aup-server ] || { echo 'refusing: --host must be aup-server' >&2; exit 1; }

ssh_bin=${TCOMPUTE_SSH_BIN:-ssh}
remote_hostname=$($ssh_bin "$host" hostname | tr -d '\r\n')
[ "$remote_hostname" = aup-test-01 ] || { echo "refusing unexpected remote hostname: $remote_hostname" >&2; exit 1; }

incus_version=$(read_version INCUS_APT_VERSION)
uidmap_version=$(read_version UIDMAP_APT_VERSION)

if [ "$mode" = check ]; then
  $ssh_bin "$host" 'set -e; . /etc/os-release; printf "os=%s version=%s\n" "$ID" "$VERSION_ID"; apt-cache policy incus uidmap | sed -n "1,30p"; command -v incus || true; command -v newuidmap || true; test -c /dev/kvm && echo kvm=present'
  exit 0
fi

$ssh_bin -tt "$host" "set -e; sudo apt-get update; sudo DEBIAN_FRONTEND=noninteractive apt-get install -y incus=$incus_version incus-client uidmap=$uidmap_version qemu-system-x86; command -v incus; command -v newuidmap; command -v newgidmap; incus version"
echo 'AUP prerequisites installed and verified'
