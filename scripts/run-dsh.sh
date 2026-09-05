#!/usr/bin/env bash
# Development fallback: pin npx to dsh.version. Production uses vendor/dsh.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PIN="$(tr -d '[:space:]' < "$ROOT/dsh.version")"
exec npx -y "@deepseek-ai/dsh@${PIN}" "$@"
