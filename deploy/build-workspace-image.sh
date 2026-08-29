#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  build-workspace-image.sh --build --host aup-server --alias <new-alias>

With no arguments this command only prints help. The command refuses to replace
an existing image alias and always cleans its fixed-name disposable builder.
EOF
}

mode=help
host=
alias_name=
while (($#)); do
  case "$1" in
    --build) [[ "$mode" == help ]] || { echo 'choose --build once' >&2; exit 2; }; mode=build; shift ;;
    --host) [[ $# -ge 2 ]] || { echo '--host requires a value' >&2; exit 2; }; host=$2; shift 2 ;;
    --alias) [[ $# -ge 2 ]] || { echo '--alias requires a value' >&2; exit 2; }; alias_name=$2; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
if [[ "$mode" == help ]]; then usage; exit 0; fi
[[ "$host" == aup-server ]] || { echo 'refusing: --host must be aup-server' >&2; exit 1; }
[[ "$alias_name" =~ ^tcompute-workspace-[a-z0-9-]{1,40}$ ]] || { echo 'invalid workspace image alias' >&2; exit 1; }

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
packages="$repo_root/images/workspace/packages.txt"
versions="$repo_root/deploy/versions.env"
if grep -Ev '^[a-z0-9][a-z0-9+.-]*$' "$packages" | grep -q .; then
  echo 'invalid workspace package list' >&2
  exit 1
fi
base_fingerprint=$(sed -n 's/^UBUNTU_WORKSPACE_BASE_FINGERPRINT=\([a-f0-9]\{64\}\)$/\1/p' "$versions")
coder_version=$(sed -n 's/^CODER_VERSION=\([0-9.]\{5,16\}\)$/\1/p' "$versions")
[[ -n "$base_fingerprint" && -n "$coder_version" ]] || { echo 'workspace image pins are missing' >&2; exit 1; }

ssh_bin=${TCOMPUTE_SSH_BIN:-ssh}
scp_bin=${TCOMPUTE_SCP_BIN:-scp}
remote_hostname=$($ssh_bin "$host" hostname | tr -d '\r\n')
[[ "$remote_hostname" == aup-test-01 ]] || { echo "refusing unexpected remote hostname: $remote_hostname" >&2; exit 1; }

remote_packages=/home/aup/tashan-compute/staging/workspace-packages.txt
remote_builder=/home/aup/tashan-compute/staging/build-workspace-image-remote.sh
$scp_bin "$packages" "$host:$remote_packages"
$scp_bin "$repo_root/images/workspace/build-remote.sh" "$host:$remote_builder"
$ssh_bin -tt "$host" "bash $remote_builder $alias_name $base_fingerprint $coder_version $remote_packages"
