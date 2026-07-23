#!/usr/bin/env bash
# Sign, notarize, and staple build/yks-tool.app; produce distribution zip.
#
# Defaults align with it-ogt-pc-mac (forge.js); override via env if needed.
#   YKS_APPLE_IDENTITY   Developer ID Application identity
#   YKS_NOTARY_PROFILE   notarytool keychain profile (shared with yksmacos)
#
# Notary fallback (if profile unset / unavailable):
#   YKS_APPLE_ID + YKS_APPLE_TEAM_ID + YKS_APPLE_APP_PASSWORD
#
# Optional:
#   YKS_APP_BUNDLE       default build/yks-tool.app
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

APP_BUNDLE="${YKS_APP_BUNDLE:-build/yks-tool.app}"
ENTITLEMENTS="packaging/macos/yks-tool.entitlements"
NOTARIZE_ZIP="build/yks-tool-macos-notarize.zip"
DIST_ZIP="build/yks-tool-macos.zip"

# 与 it-ogt-pc-mac 共用；用 SHA-1 哈希避免同名证书 ambiguous
# （login.keychain 内有两条同名 Developer ID Application）
: "${YKS_APPLE_IDENTITY:=BBAB30F5901351F4F769DFEEF702BAF26CE968C4}"
: "${YKS_NOTARY_PROFILE:=com.seaskylight.yksmacos}"

fail() {
  echo "error: $*" >&2
  exit 1
}

[[ -d "${APP_BUNDLE}" ]] || fail "missing app bundle: ${APP_BUNDLE} (run ./scripts/build-darwin.sh first)"
[[ -f "${ENTITLEMENTS}" ]] || fail "missing entitlements: ${ENTITLEMENTS}"
[[ -n "${YKS_APPLE_IDENTITY}" ]] || fail "YKS_APPLE_IDENTITY is required (e.g. Developer ID Application: ...)"

if [[ -z "${YKS_NOTARY_PROFILE}" ]]; then
  if [[ -z "${YKS_APPLE_ID:-}" || -z "${YKS_APPLE_TEAM_ID:-}" || -z "${YKS_APPLE_APP_PASSWORD:-}" ]]; then
    fail "set YKS_NOTARY_PROFILE, or YKS_APPLE_ID + YKS_APPLE_TEAM_ID + YKS_APPLE_APP_PASSWORD"
  fi
fi

command -v codesign >/dev/null 2>&1 || fail "codesign not found (need macOS + Xcode CLT)"
command -v ditto >/dev/null 2>&1 || fail "ditto not found"
command -v xcrun >/dev/null 2>&1 || fail "xcrun not found"

EXECUTABLE="${APP_BUNDLE}/Contents/MacOS/yks-tool"
[[ -f "${EXECUTABLE}" ]] || fail "missing executable: ${EXECUTABLE}"

mkdir -p build

echo "Using identity: ${YKS_APPLE_IDENTITY}"
echo "Using notary profile: ${YKS_NOTARY_PROFILE:-"(apple-id env)"}"

echo "=== 1/5 codesign (Hardened Runtime) ==="
codesign --force --options runtime --timestamp \
  --entitlements "${ENTITLEMENTS}" \
  --sign "${YKS_APPLE_IDENTITY}" \
  "${EXECUTABLE}"

codesign --force --deep --options runtime --timestamp \
  --entitlements "${ENTITLEMENTS}" \
  --sign "${YKS_APPLE_IDENTITY}" \
  "${APP_BUNDLE}"

codesign -dv --verbose=2 "${APP_BUNDLE}" 2>&1 | head -n 30
codesign --verify --deep --strict --verbose=2 "${APP_BUNDLE}"

echo "=== 2/5 zip for notarization ==="
rm -f "${NOTARIZE_ZIP}"
ditto -c -k --keepParent "${APP_BUNDLE}" "${NOTARIZE_ZIP}"

echo "=== 3/5 notarytool submit ==="
if [[ -n "${YKS_NOTARY_PROFILE}" ]]; then
  xcrun notarytool submit "${NOTARIZE_ZIP}" \
    --keychain-profile "${YKS_NOTARY_PROFILE}" \
    --wait
else
  xcrun notarytool submit "${NOTARIZE_ZIP}" \
    --apple-id "${YKS_APPLE_ID}" \
    --team-id "${YKS_APPLE_TEAM_ID}" \
    --password "${YKS_APPLE_APP_PASSWORD}" \
    --wait
fi

echo "=== 4/5 stapler staple ==="
xcrun stapler staple "${APP_BUNDLE}"
xcrun stapler validate "${APP_BUNDLE}"

echo ""
echo "=== 5/5 distribution zip ==="
rm -f "${DIST_ZIP}"
ditto -c -k --keepParent "${APP_BUNDLE}" "${DIST_ZIP}"

echo ""
echo "=== Local Gatekeeper assess (best-effort) ==="
if spctl --assess -vv --type execute "${APP_BUNDLE}" 2>&1; then
  echo "spctl assess: ok"
else
  echo "warning: spctl assess reported an issue (re-check on a clean Mac)" >&2
fi

echo ""
echo "Sign & notarize complete:"
echo "  ${APP_BUNDLE}  (stapled)"
echo "  ${DIST_ZIP}"
echo "  ${NOTARIZE_ZIP}  (notarization upload artifact; safe to delete)"
