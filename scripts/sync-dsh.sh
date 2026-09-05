#!/usr/bin/env bash
# Materialize official Node + @deepseek-ai/dsh@pin into vendor/dsh.
# Run on the target OS/arch. Output is gitignored.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PIN="${1:-}"
if [[ -z "$PIN" ]]; then
  PIN="$(tr -d '[:space:]' < "$ROOT/dsh.version")"
fi
if [[ -z "$PIN" ]]; then
  echo "dsh.version is empty" >&2
  exit 1
fi

NODE_VERSION="${NODE_VERSION:-24.18.0}"
DEST="${DSH_VENDOR_DIR:-$ROOT/vendor/dsh}"
TMP="$(mktemp -d)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
  Darwin) node_os=darwin ;;
  Linux) node_os=linux ;;
  MINGW*|MSYS*|CYGWIN*) node_os=win ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac
case "$arch" in
  arm64|aarch64) node_arch=arm64 ;;
  x86_64|amd64) node_arch=x64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

if [[ "$node_os" == "win" ]]; then
  node_archive="node-v${NODE_VERSION}-${node_os}-${node_arch}.zip"
  node_url="https://nodejs.org/dist/v${NODE_VERSION}/${node_archive}"
else
  node_archive="node-v${NODE_VERSION}-${node_os}-${node_arch}.tar.gz"
  node_url="https://nodejs.org/dist/v${NODE_VERSION}/${node_archive}"
fi

echo "syncing @deepseek-ai/dsh@${PIN} + node ${NODE_VERSION} (${node_os}-${node_arch})"
rm -rf "$DEST"
mkdir -p "$DEST"
curl -fsSL "$node_url" -o "$TMP/$node_archive"

if [[ "$node_os" == "win" ]]; then
  extracted="$TMP/node-v${NODE_VERSION}-${node_os}-${node_arch}"
  if command -v unzip >/dev/null; then
    unzip -q "$TMP/$node_archive" -d "$TMP"
  else
    python -c "import zipfile,sys; zipfile.ZipFile(sys.argv[1]).extractall(sys.argv[2])" "$TMP/$node_archive" "$TMP"
  fi
  cp "$extracted/node.exe" "$DEST/node.exe"
else
  tar -xzf "$TMP/$node_archive" -C "$TMP"
  extracted="$TMP/node-v${NODE_VERSION}-${node_os}-${node_arch}"
  cp "$extracted/bin/node" "$DEST/node"
  chmod +x "$DEST/node"
fi

# Use the host npm to install the pinned CLI into DEST.
# The bundled node binary is what the desktop shell execs at runtime.
npm install --prefix "$DEST" --omit=dev --no-fund --no-audit "@deepseek-ai/dsh@${PIN}"

if [[ ! -f "$DEST/node_modules/@deepseek-ai/dsh/lib/bin.js" ]]; then
  echo "npm install did not produce @deepseek-ai/dsh/lib/bin.js" >&2
  exit 1
fi

printf '%s\n' "$PIN" > "$DEST/VERSION"
bash "$ROOT/scripts/prune-dsh.sh" "$DEST"
echo "runtime ready at $DEST"
