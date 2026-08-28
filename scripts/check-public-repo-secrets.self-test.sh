#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT

expect_failure() {
  local expected=$1
  local output
  if output=$(cd "$repo_root" && go run ./scripts/check-public-repo-secrets --root "$fixture" 2>&1); then
    echo "secret gate accepted leak" >&2
    exit 1
  fi
  grep -Fq "$expected" <<<"$output" || {
    echo "expected '$expected', got: $output" >&2
    exit 1
  }
}

printf '%s%s\n' '-----BEGIN RSA PRIVATE ' 'KEY-----' >"$fixture/private-key.txt"
expect_failure 'private-key.txt: private key material'
rm "$fixture/private-key.txt"

printf '%s%s\n' 'Authorization: Bea' 'rer eyJhbGciOiJFZERTQSJ9.real-production-token' >"$fixture/token.txt"
expect_failure 'token.txt: bearer token'
rm "$fixture/token.txt"

printf '%s%s\n' 'DATABASE_URL=post' 'gres://real_user:real_password@db.internal/compute' >"$fixture/database.env"
expect_failure 'database.env: credential-bearing database URL'
rm "$fixture/database.env"

printf '%s%s\n' 'ADMIN_PASS' 'WORD=RealProductionPassword2026' >"$fixture/password.env"
expect_failure 'password.env: password assignment'
rm "$fixture/password.env"

printf '%s\n' 'DATABASE_URL=' 'ADMIN_PASSWORD=[REDACTED]' 'Authorization: Bearer fixture-token' >"$fixture/redacted.example"
(cd "$repo_root" && go run ./scripts/check-public-repo-secrets --root "$fixture") >/dev/null

echo "check-public-repo-secrets self-test: PASS"
