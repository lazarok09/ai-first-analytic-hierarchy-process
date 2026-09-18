#!/usr/bin/env bash
# Generate GitHub Release notes from goreleaser/chglog (changelog.yml).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! command -v chglog >/dev/null 2>&1; then
  echo "chglog not found; install with:" >&2
  echo "  go install github.com/goreleaser/chglog/cmd/chglog@v0.7.4" >&2
  exit 1
fi

OUT="${1:-.release-notes.md}"
chglog format -t release -o "$OUT"
echo "Wrote $OUT"
