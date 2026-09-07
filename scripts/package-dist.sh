#!/usr/bin/env bash
# Build people-facing release artifacts:
#   macOS  — ULMO DMG (signed/notarized when Apple secrets are present)
#   Windows — zip of the standalone exe
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

if [[ "$os" == "Darwin" ]]; then
  app="$BIN_DIR/${APP_NAME}.app"
  if [[ ! -d "$app" ]]; then
    echo "missing $app; run wails3 package first" >&2
    exit 1
  fi

  if [[ -n "${MACOS_CERTIFICATE_P12:-}" ]]; then
    bash "$ROOT/scripts/macos-sign-notarize.sh" "$app"
  fi

  dmg_asset="${APP_NAME}-darwin-${goarch}.dmg"
  mkdir -p "$stage/dmg"
  ditto "$app" "$stage/dmg/${APP_NAME}.app"
  ln -s /Applications "$stage/dmg/Applications"
  if [[ -z "${MACOS_CERTIFICATE_P12:-}" ]]; then
    printf '%s\n' '若提示“无法打开 / 已损坏”：按住 Control 点图标选“打开”，或到 系统设置 → 隐私与安全性 点“仍要打开”。' \
      > "$stage/dmg/打开说明.txt"
  fi
  rm -f "$BIN_DIR/$dmg_asset"
  hdiutil create \
    -volname "$APP_NAME" \
    -srcfolder "$stage/dmg" \
    -ov \
    -format ULMO \
    -o "$BIN_DIR/$dmg_asset" >/dev/null

  if [[ -n "${MACOS_CERTIFICATE_P12:-}" ]]; then
    bash "$ROOT/scripts/macos-sign-notarize.sh" "$app" "$BIN_DIR/$dmg_asset"
  fi
  echo "$BIN_DIR/$dmg_asset"

elif [[ "$os" == MINGW* || "$os" == MSYS* || "$os" == CYGWIN* || "$os" == Windows_NT ]]; then
  exe="$BIN_DIR/${APP_NAME}.exe"
  if [[ ! -f "$exe" ]]; then
    echo "missing $exe; build the Windows binary first" >&2
    exit 1
  fi
  inner="${APP_NAME}.exe"
  cp "$exe" "$stage/$inner"
  asset="${APP_NAME}-windows-${goarch}.zip"
  out="$BIN_DIR/$asset"
  rm -f "$out"
  if command -v zip >/dev/null; then
    (cd "$stage" && zip -9 -q "$out" "$inner")
  else
    py=python3
    command -v python3 >/dev/null || py=python
    "$py" -c "
import zipfile, sys
with zipfile.ZipFile(sys.argv[1], 'w', zipfile.ZIP_DEFLATED, compresslevel=9) as z:
    z.write(sys.argv[2], sys.argv[3])
" "$out" "$stage/$inner" "$inner"
  fi
  echo "$out"
else
  echo "package-dist: unsupported OS $os" >&2
  exit 1
fi
