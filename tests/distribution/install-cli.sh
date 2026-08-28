#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/../.." && pwd)
installer="$repo_root/skill/tashan-compute/scripts/install-cli.sh"
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT

hash_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    sha256sum "$1" | awk '{print $1}'
  fi
}

make_fixture() {
  local version=$1
  local platform=$2
  local mode=${3:-valid}
  local root="$test_root/release-$version-$mode"
  local release="$root/v$version"
  local top="tcompute-v$version-$platform"
  local package="$root/staging/$top"
  local asset="$top.tar.gz"
  mkdir -p "$package/bin" "$release"
  printf '%s\n' "$version" >"$package/VERSION"
  if [ "$mode" = smoke-fail ]; then
    printf '%s\n' '#!/bin/sh' 'exit 9' >"$package/bin/tcompute"
  else
    printf '%s\n' '#!/bin/sh' 'case "${1-}" in' "  --version) printf '%s\\n' '$version' ;;" "  '') printf '%s\\n' 'Usage: tcompute' ;;" 'esac' >"$package/bin/tcompute"
  fi
  chmod +x "$package/bin/tcompute"
  if [ "$mode" = symlink ]; then
    rm "$package/bin/tcompute"
    ln -s /bin/sh "$package/bin/tcompute"
  fi
  if [ "$mode" = extra ]; then
    mkdir -p "$root/staging/unexpected"
    printf 'unexpected\n' >"$root/staging/unexpected/file"
    tar -czf "$release/$asset" -C "$root/staging" "$top" unexpected
  else
    tar -czf "$release/$asset" -C "$root/staging" "$top"
  fi
  local digest
  digest=$(hash_file "$release/$asset")
  if [ "$mode" = bad-checksum ]; then
    digest=$(printf '0%.0s' {1..64})
  fi
  if [ "$mode" = missing-checksum ]; then
    printf '%s  other-%s\n' "$digest" "$asset" >"$release/SHA256SUMS"
  else
    printf '%s  %s\n' "$digest" "$asset" >"$release/SHA256SUMS"
  fi
  printf '%s\n' "$release"
}

new_home() {
  local label=$1
  local home="$test_root/home-$label"
  mkdir -p "$home/tmp" "$home/bin"
  printf '%s\n' "$home"
}

run_installer() {
  local home=$1
  local release=$2
  shift 2
  HOME="$home" XDG_DATA_HOME="$home/data" TMPDIR="$home/tmp" \
    TCOMPUTE_BIN_DIR="$home/bin" TCOMPUTE_INSTALL_TESTING=1 \
    TCOMPUTE_INSTALL_PLATFORM=darwin-arm64 TCOMPUTE_RELEASE_BASE_URL="file://$release" \
    bash "$installer" "$@"
}

assert_fails_with() {
  local home=$1
  local release=$2
  local expected=$3
  shift 3
  local output
  if output=$(run_installer "$home" "$release" "$@" 2>&1); then
    echo "expected installer failure: $expected" >&2
    exit 1
  fi
  if ! grep -Fq "$expected" <<<"$output"; then
    echo "expected '$expected', got: $output" >&2
    exit 1
  fi
}

version=0.1.0-alpha.1
platform=darwin-arm64
valid_release=$(make_fixture "$version" "$platform")
home=$(new_home valid)

usage=$(run_installer "$home" "$valid_release")
grep -Fq 'Usage:' <<<"$usage"
[ ! -e "$home/data" ]

assert_fails_with "$home" "$valid_release" 'version must be semver' --install --version ../../tmp/pwn

run_installer "$home" "$valid_release" --install >/dev/null
target="$home/bin/tcompute"
[ -L "$target" ]
[ "$($target --version)" = "$version" ]
run_installer "$home" "$valid_release" --install | grep -Fq 'already installed'

unmanaged_home=$(new_home unmanaged)
printf '%s\n' '#!/bin/sh' 'echo user-owned' >"$unmanaged_home/bin/tcompute"
chmod +x "$unmanaged_home/bin/tcompute"
assert_fails_with "$unmanaged_home" "$valid_release" 'refusing to replace unmanaged tcompute' --install
grep -Fq user-owned "$unmanaged_home/bin/tcompute"

for case_name in bad-checksum missing-checksum extra symlink smoke-fail; do
  release=$(make_fixture "$version" "$platform" "$case_name")
  case_home=$(new_home "$case_name")
  case "$case_name" in
    bad-checksum) message='checksum verification failed' ;;
    missing-checksum) message='checksum entry not found' ;;
    extra) message='invalid archive layout' ;;
    symlink) message='archive links are not allowed' ;;
    smoke-fail) message='installed CLI smoke test failed' ;;
  esac
  assert_fails_with "$case_home" "$release" "$message" --install
  [ ! -e "$case_home/bin/tcompute" ]
done

echo "install-cli distribution test: PASS"
