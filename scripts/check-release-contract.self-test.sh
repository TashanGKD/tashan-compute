#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT

copy_fixture() {
  rm -rf "$fixture/release" "$fixture/skill"
  mkdir -p "$fixture/release" "$fixture/skill/tashan-compute"
  cp "$repo_root/release/cli-release.json" "$fixture/release/cli-release.json"
  cp "$repo_root/skill/tashan-compute/release.json" "$fixture/skill/tashan-compute/release.json"
}

expect_failure() {
  local expected=$1
  local output
  if output=$(cd "$repo_root" && go run ./scripts/check-release-contract --root "$fixture" 2>&1); then
    echo "release self-test accepted drift" >&2
    exit 1
  fi
  grep -Fq "$expected" <<<"$output" || {
    echo "expected '$expected', got: $output" >&2
    exit 1
  }
}

copy_fixture
perl -0pi -e 's/TashanGKD\/tashan-compute/TashanGKD\/wrong-repository/' "$fixture/skill/tashan-compute/release.json"
expect_failure 'release metadata drift'

copy_fixture
perl -0pi -e 's/tcompute-v0\.1\.0-alpha\.1-linux-x64/tcompute-wrong-linux-x64/' "$fixture/release/cli-release.json"
expect_failure 'asset name mismatch: linux-x64'

echo "check-release-contract self-test: PASS"
