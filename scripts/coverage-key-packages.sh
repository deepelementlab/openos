#!/usr/bin/env bash
# Key package coverage: either summarize an existing coverage.out (set COVERAGE_FILE) or run focused tests.
# Usage from repo root (directory containing go.mod):
#   bash scripts/coverage-key-packages.sh
#   COVERAGE_FILE=coverage.out bash scripts/coverage-key-packages.sh   # after go test -coverprofile=coverage.out ./...
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ -n "${COVERAGE_FILE:-}" ]]; then
  OUT="$COVERAGE_FILE"
  if [[ ! -f "$OUT" ]]; then
    echo "coverage file not found: $OUT" >&2
    exit 1
  fi
  echo "=== Using existing $OUT ==="
else
  OUT="${COVERAGE_KEY_OUT:-coverage.key.out}"
  PKGS=(
    "./cmd/aos/..."
    "./internal/builder/..."
    "./pkg/runtime/facade/..."
  )
  go test "${PKGS[@]}" -coverprofile="$OUT" -covermode=atomic
  echo "=== Total for selected packages ==="
  go tool cover -func="$OUT" | tail -n1
fi

echo "=== Key packages (line coverage by file) ==="
go tool cover -func="$OUT" | grep -E 'github.com/agentos/aos/cmd/aos/|github.com/agentos/aos/internal/builder/|github.com/agentos/aos/pkg/runtime/facade/' || true
