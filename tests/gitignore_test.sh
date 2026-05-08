#!/usr/bin/env bash

set -u

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root" || exit 1

failures=0

assert_ignored() {
  local path="$1"
  if git check-ignore -q -- "$path"; then
    printf 'ok ignored: %s\n' "$path"
    return
  fi

  printf 'FAIL expected ignored: %s\n' "$path"
  failures=1
}

assert_not_ignored() {
  local path="$1"
  if git check-ignore -q -- "$path"; then
    printf 'FAIL expected tracked: %s\n' "$path"
    git check-ignore -v -- "$path" || true
    failures=1
    return
  fi

  printf 'ok tracked: %s\n' "$path"
}

assert_ignored ".DS_Store"
assert_ignored "coverage.out"
assert_ignored "coverage.txt"
assert_ignored "pkg.coverprofile"
assert_ignored "bin/exa-mcp-manager"
assert_ignored "build/output"
assert_ignored "dist/exa-mcp-manager"
assert_ignored ".env.local"
assert_ignored "tmp/run.log"
assert_ignored "debug.log"
assert_ignored ".cache/go-build"

assert_not_ignored "docs/exa-mcp-manager-spec.md"
assert_not_ignored "testdata/codex/config.toml"
assert_not_ignored "internal/exa/url.go"
assert_not_ignored "fixtures/exa_keys.txt"

if [ "$failures" -ne 0 ]; then
  exit 1
fi
