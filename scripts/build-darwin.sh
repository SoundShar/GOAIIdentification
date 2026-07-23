#!/usr/bin/env bash
# 构建 macOS Universal yks-tool.app（未签名，供开发联调）
# 产物：build/yks-tool.app（组装完成后删除 lipo 中间二进制）
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

APP_NAME="yks-tool"
APP_BUNDLE="build/${APP_NAME}.app"
VERSION="${YKS_VERSION:-1.0.0}"
PACKAGING_DIR="packaging/macos"
ICON_SRC="assets/icon.png"
ICNS_OUT="${PACKAGING_DIR}/AppIcon.icns"

export MACOSX_DEPLOYMENT_TARGET=12.0
export CGO_ENABLED=1

chmod +x scripts/download-deps-darwin.sh
./scripts/download-deps-darwin.sh

mkdir -p build

generate_app_icon() {
  if [[ ! -f "$ICON_SRC" ]]; then
    echo "error: missing icon source: $ICON_SRC" >&2
    exit 1
  fi
  if ! command -v sips >/dev/null 2>&1 || ! command -v iconutil >/dev/null 2>&1; then
    if [[ -f "$ICNS_OUT" ]]; then
      echo "warning: sips/iconutil unavailable, using existing $ICNS_OUT"
      return 0
    fi
    echo "error: sips/iconutil required to generate AppIcon.icns" >&2
    exit 1
  fi

  local tmp_dir iconset work_png
  tmp_dir="$(mktemp -d)"
  iconset="${tmp_dir}/AppIcon.iconset"
  work_png="${tmp_dir}/icon-1024.png"
  mkdir -p "$iconset"

  # 统一为 72 DPI，避免源图高 DPI 导致 iconutil 报 Invalid Iconset
  sips -s format png -z 1024 1024 "$ICON_SRC" --out "$work_png" >/dev/null
  sips -s dpiWidth 72 -s dpiHeight 72 "$work_png" >/dev/null

  local size
  for size in 16 32 128 256 512; do
    sips -z "$size" "$size" "$work_png" --out "${iconset}/icon_${size}x${size}.png" >/dev/null
    sips -z $((size * 2)) $((size * 2)) "$work_png" --out "${iconset}/icon_${size}x${size}@2x.png" >/dev/null
    sips -s dpiWidth 72 -s dpiHeight 72 "${iconset}/icon_${size}x${size}.png" >/dev/null
    sips -s dpiWidth 72 -s dpiHeight 72 "${iconset}/icon_${size}x${size}@2x.png" >/dev/null
  done

  xattr -cr "$iconset" 2>/dev/null || true
  iconutil -c icns "$iconset" -o "$ICNS_OUT"
  rm -rf "$tmp_dir"
  echo "Generated $ICNS_OUT"
}

echo "Building ${APP_NAME}-darwin-arm64 (MACOSX_DEPLOYMENT_TARGET=${MACOSX_DEPLOYMENT_TARGET})..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o "build/${APP_NAME}-darwin-arm64" .

echo "Building ${APP_NAME}-darwin-amd64..."
if [[ "$(uname -m)" == "arm64" ]]; then
  GOOS=darwin GOARCH=amd64 CC="clang -arch x86_64" go build -ldflags="-s -w" -o "build/${APP_NAME}-darwin-amd64" .
else
  GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o "build/${APP_NAME}-darwin-amd64" .
fi

echo "Creating universal binary with lipo..."
lipo -create \
  -output "build/${APP_NAME}-darwin-universal" \
  "build/${APP_NAME}-darwin-arm64" \
  "build/${APP_NAME}-darwin-amd64"

generate_app_icon

echo "Assembling ${APP_BUNDLE}..."
rm -rf "$APP_BUNDLE"
mkdir -p "${APP_BUNDLE}/Contents/MacOS"
mkdir -p "${APP_BUNDLE}/Contents/Resources"

cp "${PACKAGING_DIR}/Info.plist" "${APP_BUNDLE}/Contents/Info.plist"
cp "build/${APP_NAME}-darwin-universal" "${APP_BUNDLE}/Contents/MacOS/${APP_NAME}"
cp "$ICNS_OUT" "${APP_BUNDLE}/Contents/Resources/AppIcon.icns"
chmod +x "${APP_BUNDLE}/Contents/MacOS/${APP_NAME}"

if command -v plutil >/dev/null 2>&1; then
  plutil -replace CFBundleShortVersionString -string "$VERSION" "${APP_BUNDLE}/Contents/Info.plist"
  plutil -replace CFBundleVersion -string "$VERSION" "${APP_BUNDLE}/Contents/Info.plist"
  plutil -lint "${APP_BUNDLE}/Contents/Info.plist"
fi

echo ""
echo "=== Verification ==="
lipo -info "${APP_BUNDLE}/Contents/MacOS/${APP_NAME}"
if command -v defaults >/dev/null 2>&1; then
  echo "LSMinimumSystemVersion=$(defaults read "${ROOT_DIR}/${APP_BUNDLE}/Contents/Info" LSMinimumSystemVersion 2>/dev/null || true)"
  echo "CFBundleIdentifier=$(defaults read "${ROOT_DIR}/${APP_BUNDLE}/Contents/Info" CFBundleIdentifier 2>/dev/null || true)"
fi

echo "Removing lipo intermediate binaries..."
rm -f \
  "build/${APP_NAME}-darwin-arm64" \
  "build/${APP_NAME}-darwin-amd64" \
  "build/${APP_NAME}-darwin-universal"

echo ""
echo "Build complete (unsigned .app for local use):"
echo "  ${APP_BUNDLE}"
echo ""
echo "Run: open ${APP_BUNDLE}"
echo "Sign & notarize (production): ./scripts/sign-notarize-darwin.sh"
