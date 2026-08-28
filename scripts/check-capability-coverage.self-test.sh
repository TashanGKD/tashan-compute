#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
fixture_root=$(mktemp -d)
trap 'rm -rf "$fixture_root"' EXIT

copy_fixture() {
  rm -rf "$fixture_root/capabilities" "$fixture_root/internal" "$fixture_root/skill"
  mkdir -p "$fixture_root/capabilities" "$fixture_root/internal/cli" "$fixture_root/skill/tashan-compute"
  cp "$repo_root/capabilities/manifest.json" "$fixture_root/capabilities/manifest.json"
  cp "$repo_root/internal/cli/bindings.json" "$fixture_root/internal/cli/bindings.json"
  cp "$repo_root/skill/tashan-compute/capability-references.json" "$fixture_root/skill/tashan-compute/capability-references.json"
}

run_gate_expect_failure() {
  local expected=$1
  local output
  if output=$(cd "$repo_root" && go run ./scripts/check-capability-coverage --root "$fixture_root" 2>&1); then
    echo "self-test failed: gate accepted invalid fixture" >&2
    exit 1
  fi
  if ! grep -Fq "$expected" <<<"$output"; then
    echo "self-test failed: expected '$expected', got: $output" >&2
    exit 1
  fi
}

copy_fixture
perl -0pi -e 's/^\s*"admin\.user\.create"[^\n]*\n//m' "$fixture_root/internal/cli/bindings.json"
run_gate_expect_failure "missing CLI binding: admin.user.create"

copy_fixture
perl -0pi -e 's/"audit\.list"/"audit.lits"/' "$fixture_root/skill/tashan-compute/capability-references.json"
run_gate_expect_failure "unknown Skill capability: audit.lits"

echo "check-capability-coverage self-test: PASS"
