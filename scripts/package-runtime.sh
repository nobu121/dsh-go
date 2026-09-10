#!/usr/bin/env bash
# Zip vendor/dsh as dsh-runtime-<os>-<arch>.zip (top-level node + npm tree)
# and write the runtime.json the shell reads to learn the current dsh version.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="${BIN_DIR:-$ROOT/bin}"
DEST="${DSH_VENDOR_DIR:-$ROOT/vendor/dsh}"
os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
  Darwin) goos=darwin ;;
  Linux) goos=wsl ;;
  MINGW*|MSYS*|CYGWIN*|Windows_NT) goos=windows ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac
case "$arch" in
  arm64|aarch64) goarch=arm64 ;;
  x86_64|amd64) goarch=amd64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

if [[ ! -x "$DEST/node" && ! -f "$DEST/node.exe" ]]; then
  echo "missing $DEST; run scripts/sync-dsh.sh first" >&2
  exit 1
fi
if [[ ! -f "$DEST/node_modules/@deepseek-ai/dsh/lib/bin.js" ]]; then
  echo "missing @deepseek-ai/dsh in $DEST" >&2
  exit 1
fi

mkdir -p "$BIN_DIR"
asset="dsh-runtime-${goos}-${goarch}.zip"
out="$BIN_DIR/$asset"
rm -f "$out"
(
  cd "$DEST"
  if command -v zip >/dev/null; then
    zip -9 -qr -y "$out" . -x '*.DS_Store' -x '*.DS_Store/*'
  else
    python3 -c "import shutil,sys; shutil.make_archive(sys.argv[1], 'zip', sys.argv[2])" \
      "${out%.zip}" "$DEST"
  fi
)

ver="$(tr -d '[:space:]' < "$DEST/VERSION" 2>/dev/null || true)"
if [[ -z "$ver" ]]; then
  echo "missing $DEST/VERSION; run scripts/sync-dsh.sh first" >&2
  exit 1
fi
cat > "$BIN_DIR/runtime.json" <<EOF
{
  "version": "${ver}",
  "updated": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

echo "$out"
echo "$BIN_DIR/runtime.json"
