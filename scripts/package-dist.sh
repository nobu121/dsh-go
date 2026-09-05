#!/usr/bin/env bash
# Build shell-only release artifacts:
#   macOS  — ULMO DMG for people, max-deflate zip for the updater
#   Windows — max-deflate zip (portable folder, also the updater asset)
# Runtime zips are produced separately by scripts/package-runtime.sh.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="${BIN_DIR:-$ROOT/bin}"
APP_NAME="${APP_NAME:-dsh-go}"
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

write_zip_max() {
  local src_dir="$1" entry="$2" out="$3"
  rm -f "$out"
  (
    cd "$src_dir"
    if command -v zip >/dev/null; then
      zip -9 -qr -y "$out" "$entry"
    else
      python3 -c "import shutil,sys; shutil.make_archive(sys.argv[1], 'zip', sys.argv[2], sys.argv[3])" \
        "${out%.zip}" "$src_dir" "$entry"
    fi
  )
}

if [[ "$os" == "Darwin" ]]; then
  app="$BIN_DIR/${APP_NAME}.app"
  if [[ ! -d "$app" ]]; then
    echo "missing $app; run wails3 package first" >&2
    exit 1
  fi

  zip_asset="${APP_NAME}-darwin-${goarch}.zip"
  # Single top-level entry (.app) for the Wails updater.
  if ditto -c -k --keepParent --zlibCompressionLevel 9 "$app" "$BIN_DIR/$zip_asset" 2>/dev/null; then
    :
  else
    write_zip_max "$BIN_DIR" "${APP_NAME}.app" "$BIN_DIR/$zip_asset"
  fi
  echo "$BIN_DIR/$zip_asset"

  dmg_asset="${APP_NAME}-darwin-${goarch}.dmg"
  mkdir -p "$stage/dmg"
  ditto "$app" "$stage/dmg/${APP_NAME}.app"
  ln -s /Applications "$stage/dmg/Applications"
  rm -f "$BIN_DIR/$dmg_asset"
  hdiutil create \
    -volname "$APP_NAME" \
    -srcfolder "$stage/dmg" \
    -ov \
    -format ULMO \
    -o "$BIN_DIR/$dmg_asset" >/dev/null
  echo "$BIN_DIR/$dmg_asset"

elif [[ "$os" == MINGW* || "$os" == MSYS* || "$os" == CYGWIN* || "$os" == Windows_NT ]]; then
  exe="$BIN_DIR/${APP_NAME}.exe"
  if [[ ! -f "$exe" ]]; then
    echo "missing $exe; build the Windows binary first" >&2
    exit 1
  fi
  mkdir -p "$stage/${APP_NAME}"
  cp "$exe" "$stage/${APP_NAME}/${APP_NAME}.exe"
  asset="${APP_NAME}-windows-${goarch}.zip"
  write_zip_max "$stage" "$APP_NAME" "$BIN_DIR/$asset"
  echo "$BIN_DIR/$asset"
else
  echo "package-dist: unsupported OS $os" >&2
  exit 1
fi
