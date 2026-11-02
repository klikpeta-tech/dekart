#!/bin/sh
# Regenerate the committed README.md of every Go dark library from its doc comments and examples.
set -eu
cd "$(dirname "$0")/.."
[ -d src/server/dark ] || exit 0
go tool gomarkdoc --output '{{.Dir}}/README.md' ./src/server/dark/...
