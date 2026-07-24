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
ICON_SRC="assets/icon.icns"
ICNS_OUT="${PACKAGING_DIR}/AppIcon.icns"

export MACOSX_DEPLOYMENT_TARGET=12.0
export CGO_ENABLED=1

chmod +x scripts/download-deps-darwin.sh
./scripts/download-deps-darwin.sh

mkdir -p build

copy_app_icon() {
  if [[ ! -f "$ICON_SRC" ]]; then
    echo "error: missing icon source: $ICON_SRC" >&2
    exit 1
  fi
  mkdir -p "$PACKAGING_DIR"
  cp -f "$ICON_SRC" "$ICNS_OUT"
  echo "Copied $ICON_SRC -> $ICNS_OUT"
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

copy_app_icon

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
