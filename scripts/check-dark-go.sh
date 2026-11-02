#!/bin/sh
# Isolation, no-mock and documentation checks for Go dark libraries under src/server/dark.
# Policy lives in AGENTS.md and skills/dark-library/SKILL.md; the allow/deny list is kept only here.
set -eu
cd "$(dirname "$0")/.."
root=src/server/dark
[ -d "$root" ] || exit 0

fail() { echo "check-dark-go: $1" >&2; exit 1; }

# Direct imports of packages and their tests (never -deps: fmt imports os and stays allowed).
imports=$(go list -f '{{.ImportPath}} {{join .Imports " "}} {{join .TestImports " "}} {{join .XTestImports " "}}' ./src/server/dark/...)
denied='^(os|os/exec|net|net/.*|database/sql|log|syscall|unsafe|io/ioutil|math/rand|math/rand/v2|crypto/rand|cloud\.google\.com/.*|github\.com/rs/zerolog)$'
echo "$imports" | while read -r package rest; do
  for imported in $rest; do
    if echo "$imported" | grep -Eq "$denied"; then fail "$package imports forbidden package $imported"; fi
    # Other code of this module is reachable only through another dark library's public entry.
    case "$imported" in
      dekart/src/server/dark/*) ;;
      dekart/*) fail "$package imports app package $imported" ;;
    esac
  done
done

# Direct clock, timer, once-lazy and init side effects. Callers inject time and I/O.
if grep -rnE 'time\.(Now|After|AfterFunc|Tick|NewTimer|NewTicker|Sleep)\(|sync\.Once|^func init\(\)' --include='*.go' "$root"; then
  fail "clock, timer, sync.Once or init() used in a dark library"
fi
# Test doubles are not allowed: contract tests run real inputs through the public API.
if grep -rnE 'httptest|sqlmock|gomock|testify/mock|mockery' --include='*.go' "$root"; then
  fail "mocks used in a dark library test"
fi
# Tests exercise the public interface only, so they live in the external package.
for file in $(find "$root" -name '*_test.go'); do
  grep -Eq '^package [a-z0-9]+_test$' "$file" || fail "$file must declare package <name>_test"
done
# Every library documents itself in doc.go and ships a generated README.
for dir in $(find "$root" -mindepth 1 -maxdepth 1 -type d); do
  [ -f "$dir/doc.go" ] || fail "$dir has no doc.go"
  [ -f "$dir/README.md" ] || fail "$dir has no generated README.md"
done
# README freshness: regenerate and fail on any difference.
go tool gomarkdoc --check --output '{{.Dir}}/README.md' ./src/server/dark/... || fail "README.md is stale: run scripts/dark-docs-go.sh"
