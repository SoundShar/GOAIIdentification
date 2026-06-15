#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

ORT_VERSION="1.19.2"
EMBED_DIR="embeddata"

mkdir -p "$EMBED_DIR/darwin_arm64" "$EMBED_DIR/darwin_amd64"

require_onnx_models() {
  local missing=0
  for name in yolo11.onnx face_detect.onnx face_rec.onnx; do
    if [[ ! -f "$EMBED_DIR/$name" ]]; then
      echo "Missing $EMBED_DIR/$name"
      missing=1
    fi
  done
  if [[ "$missing" -eq 1 ]]; then
    echo "Run scripts/download-deps.ps1 on Windows, or export models on macOS:"
    echo "  pip install ultralytics && python -c \"from ultralytics import YOLO; YOLO('yolo11n.pt').export(format='onnx', imgsz=640, opset=17, simplify=True)\""
    echo "Then copy yolo11n.onnx to embeddata/yolo11.onnx and download face models."
    exit 1
  fi
}

download_ort_dylib() {
  local arch="$1"
  local url="$2"
  local out_dir="$3"
  local out_file="$out_dir/libonnxruntime.dylib"
  local tmp_dir="$EMBED_DIR/ort_tmp_${arch}"

  if [[ -f "$out_file" ]]; then
    echo "ONNX Runtime $arch exists: $out_file"
    return
  fi

  echo "Downloading ONNX Runtime $ORT_VERSION ($arch)..."
  rm -rf "$tmp_dir"
  mkdir -p "$tmp_dir"
  curl -fsSL "$url" -o "$tmp_dir/ort.tgz"
  tar -xzf "$tmp_dir/ort.tgz" -C "$tmp_dir"
  cp "$tmp_dir/onnxruntime-osx-${arch}-${ORT_VERSION}/lib/libonnxruntime.dylib" "$out_file"
  rm -rf "$tmp_dir"
  echo "Saved: $out_file"
}

require_onnx_models

download_ort_dylib "arm64" \
  "https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/onnxruntime-osx-arm64-${ORT_VERSION}.tgz" \
  "$EMBED_DIR/darwin_arm64"

download_ort_dylib "x86_64" \
  "https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/onnxruntime-osx-x86_64-${ORT_VERSION}.tgz" \
  "$EMBED_DIR/darwin_amd64"

echo "Darwin embed assets ready in $EMBED_DIR/darwin_*/"
