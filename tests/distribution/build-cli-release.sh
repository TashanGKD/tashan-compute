#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/../.." && pwd)
builder="$repo_root/scripts/build-cli-release.sh"
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT

usage=$(bash "$builder")
grep -Fq 'Usage:' <<<"$usage"
[ -z "$(find "$test_root" -mindepth 1 -print -quit)" ]

mkdir -p "$test_root/coder-dist"
: >"$test_root/coder-dist/SHA256SUMS"
for platform in darwin-arm64 darwin-x64 linux-x64; do
  coder="$test_root/coder-dist/coder-$platform"
  printf '%s\n' '#!/bin/sh' 'echo Coder v2.35.6' >"$coder"
  chmod +x "$coder"
  digest=$(shasum -a 256 "$coder" | awk '{print $1}')
  printf '%s  %s\n' "$digest" "coder-$platform" >>"$test_root/coder-dist/SHA256SUMS"
done

bash "$builder" --build --output "$test_root/out" --coder-dist "$test_root/coder-dist" >/dev/null
[ -f "$test_root/out/SHA256SUMS" ]
for platform in darwin-arm64 darwin-x64 linux-x64; do
  asset="$test_root/out/tcompute-v0.2.1-$platform.tar.gz"
  [ -f "$asset" ]
  entries="$test_root/entries-$platform.txt"
  tar -tzf "$asset" >"$entries"
  grep -Fxq "tcompute-v0.2.1-$platform/bin/tcompute" "$entries"
  grep -Fxq "tcompute-v0.2.1-$platform/bin/coder" "$entries"
done

mkdir "$test_root/unpack"
case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) smoke_platform=darwin-arm64 ;;
  Darwin-x86_64) smoke_platform=darwin-x64 ;;
  Linux-x86_64) smoke_platform=linux-x64 ;;
  *) echo "unsupported release smoke platform: $(uname -s)-$(uname -m)" >&2; exit 1 ;;
esac
tar -xzf "$test_root/out/tcompute-v0.2.1-$smoke_platform.tar.gz" -C "$test_root/unpack"
[ "$($test_root/unpack/tcompute-v0.2.1-$smoke_platform/bin/tcompute --version)" = "0.2.1" ]

echo "build-cli-release distribution test: PASS"
