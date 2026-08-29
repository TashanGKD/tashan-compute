#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
script="$repo_root/deploy/build-workspace-image.sh"
packages="$repo_root/images/workspace/packages.txt"
remote_script="$repo_root/images/workspace/build-remote.sh"

usage=$(bash "$script")
grep -Fq 'Usage:' <<<"$usage"
grep -Fxq podman "$packages"
grep -Fxq postgresql "$packages"
grep -Fxq redis-server "$packages"
grep -Fxq rustc "$packages"
grep -Fq 'tcompute-boundary-check' "$remote_script"
grep -Fq 'cloud-init clean --logs --machine-id' "$remote_script"
grep -Fq 'refusing: image alias already exists' "$remote_script"

echo 'build-workspace-image self-test: PASS'
