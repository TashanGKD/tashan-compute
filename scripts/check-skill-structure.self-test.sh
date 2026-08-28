#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/skill"
cp -R "$repo_root/skill/tashan-compute" "$fixture/skill/tashan-compute"
perl -0pi -e 's/name: tashan-compute/name: Tashan_Compute/' "$fixture/skill/tashan-compute/SKILL.md"

output=
if output=$(cd "$repo_root" && go run ./scripts/check-skill-structure --root "$fixture" 2>&1); then
  echo "skill self-test accepted invalid name" >&2
  exit 1
fi
grep -Fq 'invalid Skill name' <<<"$output" || { echo "unexpected output: $output" >&2; exit 1; }

echo "check-skill-structure self-test: PASS"
