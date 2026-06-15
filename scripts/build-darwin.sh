#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

chmod +x scripts/download-deps-darwin.sh
./scripts/download-deps-darwin.sh

mkdir -p build

export CGO_ENABLED=1

echo "Building yks-tool-darwin-arm64..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o build/yks-tool-darwin-arm64 .

echo "Building yks-tool-darwin-amd64..."
if [[ "$(uname -m)" == "arm64" ]]; then
  GOOS=darwin GOARCH=amd64 CC="clang -arch x86_64" go build -ldflags="-s -w" -o build/yks-tool-darwin-amd64 .
else
  GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o build/yks-tool-darwin-amd64 .
fi

if command -v file >/dev/null 2>&1; then
  file build/yks-tool-darwin-arm64
  file build/yks-tool-darwin-amd64
fi

echo "Build complete:"
echo "  build/yks-tool-darwin-arm64  (Apple Silicon)"
echo "  build/yks-tool-darwin-amd64  (Intel Mac)"
