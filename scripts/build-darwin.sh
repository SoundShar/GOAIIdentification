#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p build

GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o build/myapp .

echo "Build complete: build/myapp"
