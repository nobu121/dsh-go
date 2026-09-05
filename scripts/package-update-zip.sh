#!/usr/bin/env bash
# Back-compat wrapper. Prefer scripts/package-dist.sh.
set -euo pipefail
exec "$(cd "$(dirname "$0")" && pwd)/package-dist.sh" "$@"
