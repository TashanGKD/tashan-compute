#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
cp "$repo_root/scripts/verify-foundation.sh" "$fixture/verify-foundation.sh"
perl -0pi -e 's/^run_gate "check-public-repo-secrets\.self-test\.sh".*\n//m' "$fixture/verify-foundation.sh"

output=
if output=$(bash "$fixture/verify-foundation.sh" --self-check 2>&1); then
  echo "verify self-test accepted missing required gate" >&2
  exit 1
fi
grep -Fq 'required gate missing: check-public-repo-secrets.self-test.sh' <<<"$output" || {
  echo "unexpected verifier output: $output" >&2
  exit 1
}

echo "verify-foundation self-test: PASS"
