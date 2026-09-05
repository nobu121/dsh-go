#!/usr/bin/env bash
# Start the bundled runtime and wait for the authenticated dsh web URL.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="${DSH_VENDOR_DIR:-$ROOT/vendor/dsh}"
if [[ -x "$DEST/node" ]]; then
  NODE="$DEST/node"
elif [[ -f "$DEST/node.exe" ]]; then
  NODE="$DEST/node.exe"
else
  echo "bundled node missing; run scripts/sync-dsh.sh first" >&2
  exit 1
fi
BIN="$DEST/node_modules/@deepseek-ai/dsh/lib/bin.js"
if [[ ! -f "$BIN" ]]; then
  echo "bundled dsh missing; run scripts/sync-dsh.sh first" >&2
  exit 1
fi

HOME_DIR="${DSH_HOME:-$(mktemp -d)}"
export DSH_HOME="$HOME_DIR"
timeout_s="${SMOKE_TIMEOUT:-90}"
log="$(mktemp)"

"$NODE" "$BIN" --profile web --no-open --port 0 >"$log" 2>&1 &
pid=$!
cleanup() {
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
}
trap cleanup EXIT

for _ in $(seq 1 "$timeout_s"); do
  if grep -E '^dsh web: https?://' "$log" >/dev/null 2>&1; then
    echo "smoke ok"
    exit 0
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    echo "smoke failed; dsh exited early" >&2
    cat "$log" >&2
    exit 1
  fi
  sleep 1
done

echo "smoke failed; no dsh web URL within ${timeout_s}s" >&2
cat "$log" >&2
exit 1
