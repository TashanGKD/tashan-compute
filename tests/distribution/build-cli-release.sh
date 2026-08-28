#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/../.." && pwd)
builder="$repo_root/scripts/build-cli-release.sh"
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT

usage=$(bash "$builder")
grep -Fq 'Usage:' <<<"$usage"
[ -z "$(find "$test_root" -mindepth 1 -print -quit)" ]

bash "$builder" --build --output "$test_root/out" >/dev/null
[ -f "$test_root/out/SHA256SUMS" ]
for platform in darwin-arm64 darwin-x64 linux-x64; do
  asset="$test_root/out/tcompute-v0.1.0-alpha.1-$platform.tar.gz"
  [ -f "$asset" ]
  tar -tzf "$asset" | grep -Fxq "tcompute-v0.1.0-alpha.1-$platform/bin/tcompute"
done

mkdir "$test_root/unpack"
tar -xzf "$test_root/out/tcompute-v0.1.0-alpha.1-darwin-arm64.tar.gz" -C "$test_root/unpack"
[ "$($test_root/unpack/tcompute-v0.1.0-alpha.1-darwin-arm64/bin/tcompute --version)" = "0.1.0-alpha.1" ]

echo "build-cli-release distribution test: PASS"
