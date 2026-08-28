#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
metadata="$repo_root/release/cli-release.json"

usage() {
  printf '%s\n' 'Usage:' '  build-cli-release.sh --build --output <directory>' '' 'With no arguments this command only prints help.'
}

mode=help
output=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --build)
      [ "$mode" = help ] || { echo 'choose --build once' >&2; exit 1; }
      mode=build
      shift
      ;;
    --output)
      [ "$#" -ge 2 ] || { echo '--output requires a value' >&2; exit 1; }
      output=$2
      shift 2
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

if [ "$mode" = help ]; then
  usage
  exit 0
fi
[ -n "$output" ] || { echo '--output is required with --build' >&2; exit 1; }
case "$output" in
  /*) ;;
  *) output="$PWD/$output" ;;
esac
mkdir -p "$output"
[ -z "$(find "$output" -mindepth 1 -maxdepth 1 -print -quit)" ] || { echo "output directory must be empty: $output" >&2; exit 1; }

version=$(sed -n 's/^[[:space:]]*"version": "\([^"]*\)",\{0,1\}$/\1/p' "$metadata" | head -n 1)
[ -n "$version" ] || { echo 'release version is missing' >&2; exit 1; }

temporary_root=$(mktemp -d "${TMPDIR:-/tmp}/tcompute-release.XXXXXX")
trap 'rm -rf "$temporary_root"' EXIT HUP INT TERM
checksums="$output/SHA256SUMS"
: >"$checksums"

for platform in darwin-arm64 darwin-x64 linux-x64; do
  case "$platform" in
    darwin-arm64) goos=darwin; goarch=arm64 ;;
    darwin-x64) goos=darwin; goarch=amd64 ;;
    linux-x64) goos=linux; goarch=amd64 ;;
  esac
  top="tcompute-v$version-$platform"
  package="$temporary_root/$top"
  mkdir -p "$package/bin"
  printf '%s\n' "$version" >"$package/VERSION"
  (
    cd "$repo_root"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
      go build -trimpath -ldflags "-s -w -X github.com/TashanGKD/tashan-compute/internal/buildinfo.Version=$version" \
      -o "$package/bin/tcompute" ./cmd/tcompute
  )
  asset="$output/$top.tar.gz"
  tar -czf "$asset" -C "$temporary_root" "$top"
  if command -v shasum >/dev/null 2>&1; then
    digest=$(shasum -a 256 "$asset" | awk '{print $1}')
  else
    digest=$(sha256sum "$asset" | awk '{print $1}')
  fi
  printf '%s  %s\n' "$digest" "$(basename "$asset")" >>"$checksums"
  rm -rf "$package"
done

printf 'built tcompute %s release assets in %s\n' "$version" "$output"
