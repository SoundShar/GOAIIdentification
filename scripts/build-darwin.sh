#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p build

GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o build/yks-tool .

if [ -d models ]; then
  cp -R models build/models
fi

if [ -f libonnxruntime.dylib ]; then
  cp libonnxruntime.dylib build/
fi

echo "Build complete: build/yks-tool"
