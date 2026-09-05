#!/usr/bin/env bash
# Build a single-top-level zip for the Wails updater.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="${BIN_DIR:-$ROOT/bin}"
APP_NAME="${APP_NAME:-dsh-go}"
DEST="${DSH_VENDOR_DIR:-$ROOT/vendor/dsh}"
os="$(uname -s)"
arch="$(uname -m)"
case "$arch" in
  arm64|aarch64) goarch=arm64 ;;
  x86_64|amd64) goarch=amd64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

mkdir -p "$BIN_DIR"
stage="$(mktemp -d)"
cleanup() { rm -rf "$stage"; }
trap cleanup EXIT

if [[ "$os" == "Darwin" ]]; then
  app="$BIN_DIR/${APP_NAME}.app"
  if [[ ! -d "$app" ]]; then
    echo "missing $app; run wails3 package first" >&2
    exit 1
  fi
  asset="${APP_NAME}-darwin-${goarch}.zip"
  ditto -c -k --keepParent "$app" "$BIN_DIR/$asset"
elif [[ "$os" == MINGW* || "$os" == MSYS* || "$os" == CYGWIN* || "$os" == Windows_NT ]]; then
  exe="$BIN_DIR/${APP_NAME}.exe"
  if [[ ! -f "$exe" ]]; then
    echo "missing $exe; build the Windows binary first" >&2
    exit 1
  fi
  if [[ ! -d "$DEST" ]]; then
    echo "missing $DEST; run scripts/sync-dsh.sh first" >&2
    exit 1
  fi
  mkdir -p "$stage/${APP_NAME}/dsh-runtime"
  cp "$exe" "$stage/${APP_NAME}/${APP_NAME}.exe"
  cp -R "$DEST/." "$stage/${APP_NAME}/dsh-runtime/"
  asset="${APP_NAME}-windows-${goarch}.zip"
  (
    cd "$stage"
    if command -v zip >/dev/null; then
      zip -qr "$BIN_DIR/$asset" "$APP_NAME"
    else
      python -c "import shutil,sys; shutil.make_archive(sys.argv[1], 'zip', sys.argv[2], sys.argv[3])" \
        "${BIN_DIR}/${asset%.zip}" "$stage" "$APP_NAME"
    fi
  )
else
  echo "package-update-zip: unsupported OS $os" >&2
  exit 1
fi

echo "$BIN_DIR/$asset"
