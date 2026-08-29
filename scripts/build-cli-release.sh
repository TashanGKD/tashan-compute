#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
metadata="$repo_root/release/cli-release.json"

usage() {
  printf '%s\n' 'Usage:' '  build-cli-release.sh --build --output <directory> --coder-dist <verified-directory>' '' 'With no arguments this command only prints help.'
}

mode=help
output=
coder_dist=
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
    --coder-dist)
      [ "$#" -ge 2 ] || { echo '--coder-dist requires a value' >&2; exit 1; }
      coder_dist=$2
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
[ -n "$coder_dist" ] || { echo '--coder-dist is required with --build' >&2; exit 1; }
case "$output" in
  /*) ;;
  *) output="$PWD/$output" ;;
esac
mkdir -p "$output"
[ -d "$coder_dist" ] || { echo "Coder dist directory is missing: $coder_dist" >&2; exit 1; }
[ -z "$(find "$output" -mindepth 1 -maxdepth 1 -print -quit)" ] || { echo "output directory must be empty: $output" >&2; exit 1; }

version=$(sed -n 's/^[[:space:]]*"version": "\([^"]*\)",\{0,1\}$/\1/p' "$metadata" | head -n 1)
[ -n "$version" ] || { echo 'release version is missing' >&2; exit 1; }
coder_checksums="$coder_dist/SHA256SUMS"
[ -f "$coder_checksums" ] || { echo 'Coder dist SHA256SUMS is missing' >&2; exit 1; }

hash_file() {
  if command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | awk '{print $1}'; else sha256sum "$1" | awk '{print $1}'; fi
}

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
  coder_name="coder-$platform"
  coder_source="$coder_dist/$coder_name"
  [ -x "$coder_source" ] || { echo "verified Coder binary is missing: $coder_name" >&2; exit 1; }
  coder_hash=$(awk -v file="$coder_name" '$2 == file {count += 1; hash = $1} END {if (count == 1) print hash; else exit 1}' "$coder_checksums") || { echo "Coder checksum entry is missing: $coder_name" >&2; exit 1; }
  [ "$(hash_file "$coder_source")" = "$coder_hash" ] || { echo "Coder checksum mismatch: $coder_name" >&2; exit 1; }
  cp "$coder_source" "$package/bin/coder"
  chmod 0755 "$package/bin/coder"
  asset="$output/$top.tar.gz"
  tar -czf "$asset" -C "$temporary_root" "$top"
  digest=$(hash_file "$asset")
  printf '%s  %s\n' "$digest" "$(basename "$asset")" >>"$checksums"
  rm -rf "$package"
done

printf 'built tcompute %s release assets in %s\n' "$version" "$output"
