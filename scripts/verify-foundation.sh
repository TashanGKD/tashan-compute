#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
required_gates=(
  "check-capability-coverage.self-test.sh"
  "check-release-contract.self-test.sh"
  "check-public-repo-secrets.self-test.sh"
  "install-cli.sh"
  "build-cli-release.sh"
)

self_check() {
  local gate
  for gate in "${required_gates[@]}"; do
    if ! grep -Fq "run_gate \"$gate\"" "$0"; then
      echo "required gate missing: $gate" >&2
      exit 1
    fi
  done
  echo "verify-foundation self-check: PASS"
}

run_gate() {
  local name=$1
  shift
  printf '==> %s\n' "$name"
  "$@"
}

if [ "${1-}" = "--self-check" ]; then
  self_check
  exit 0
fi
if [ "$#" -ne 0 ]; then
  echo 'Usage: verify-foundation.sh [--self-check]' >&2
  exit 1
fi

cd "$repo_root"
self_check
unformatted=$(gofmt -l cmd internal migrations scripts)
[ -z "$unformatted" ] || { printf 'gofmt required:\n%s\n' "$unformatted" >&2; exit 1; }

run_gate "go-test" go test ./...
run_gate "go-vet" go vet ./...
run_gate "capability-coverage" go run ./scripts/check-capability-coverage --root .
run_gate "release-contract" go run ./scripts/check-release-contract --root .
run_gate "public-repo-secrets" go run ./scripts/check-public-repo-secrets --root .
run_gate "check-capability-coverage.self-test.sh" bash scripts/check-capability-coverage.self-test.sh
run_gate "check-release-contract.self-test.sh" bash scripts/check-release-contract.self-test.sh
run_gate "check-public-repo-secrets.self-test.sh" bash scripts/check-public-repo-secrets.self-test.sh
run_gate "install-cli.sh" bash tests/distribution/install-cli.sh
run_gate "build-cli-release.sh" bash tests/distribution/build-cli-release.sh

echo "foundation verification: PASS"
