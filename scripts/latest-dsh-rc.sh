#!/usr/bin/env bash
# Print npm's official latest for @deepseek-ai/dsh when it differs from
# dsh.version. Empty stdout = nothing to do. Alpha lives on the alpha
# dist-tag and is ignored.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PIN="$(tr -d '[:space:]' < "$ROOT/dsh.version" 2>/dev/null || true)"

latest="$(npm view @deepseek-ai/dsh version 2>/dev/null || true)"
latest="$(printf '%s' "$latest" | tr -d '[:space:]')"

if [[ -z "$latest" || "$latest" == "$PIN" ]]; then
  exit 0
fi

printf '%s\n' "$latest"
