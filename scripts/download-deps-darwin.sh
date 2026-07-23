#!/usr/bin/env bash
# 准备 macOS 构建所需的 embeddata：三个共用 ONNX 模型 + 各架构 libonnxruntime.dylib
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

ORT_VERSION="1.19.2"
EMBED_DIR="embeddata"

mkdir -p "$EMBED_DIR/darwin_arm64" "$EMBED_DIR/darwin_amd64"

python_bin() {
  # Prefer project venv (Homebrew Python is PEP 668 managed)
  if [[ -x "$ROOT_DIR/.venv/bin/python" ]]; then
    echo "$ROOT_DIR/.venv/bin/python"
  elif command -v python3 >/dev/null 2>&1; then
    echo "python3"
  elif command -v python >/dev/null 2>&1; then
    echo "python"
  else
    echo ""
  fi
}

ensure_yolo_onnx() {
  local yolo_path="$EMBED_DIR/yolo11.onnx"
  local py
  py="$(python_bin)"

  if [[ -f "$yolo_path" ]]; then
    if [[ -n "$py" ]] && "$py" -c "import onnx" >/dev/null 2>&1; then
      local opset
      opset="$("$py" -c "import onnx; m=onnx.load(r'$yolo_path'); print(m.opset_import[0].version)")"
      if [[ "$opset" == "17" ]]; then
        echo "YOLO11 model exists (opset 17): $yolo_path"
        return
      fi
      echo "YOLO11 opset $opset unsupported, re-exporting..."
    else
      echo "YOLO11 model exists: $yolo_path"
      return
    fi
  fi

  if [[ -z "$py" ]]; then
    echo "error: python3 required to export yolo11.onnx" >&2
    echo "  python3 -m venv .venv && .venv/bin/pip install ultralytics onnx" >&2
    exit 1
  fi

  if ! "$py" -c "import ultralytics" >/dev/null 2>&1; then
    echo "error: ultralytics not found in $py" >&2
    echo "  python3 -m venv .venv && .venv/bin/pip install ultralytics onnx" >&2
    exit 1
  fi

  echo "Exporting YOLO11n ONNX (opset 17, onnxruntime ${ORT_VERSION})..."
  "$py" -c "from ultralytics import YOLO; YOLO('yolo11n.pt').export(format='onnx', imgsz=640, opset=17, simplify=True)"
  if [[ ! -f "yolo11n.onnx" ]]; then
    echo "error: YOLO11 export failed" >&2
    exit 1
  fi
  cp -f "yolo11n.onnx" "$yolo_path"
  rm -f "yolo11n.onnx" "yolo11n.pt"
  echo "YOLO11 model ready: $yolo_path"
}

ensure_face_detect_onnx() {
  local path="$EMBED_DIR/face_detect.onnx"
  local url="https://github.com/opencv/opencv_zoo/raw/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx"
  if [[ -f "$path" ]]; then
    echo "Face detect model exists: $path"
    return
  fi
  echo "Downloading YuNet face detector..."
  curl -fsSL "$url" -o "$path"
  echo "Saved: $path"
}

ensure_face_rec_onnx() {
  local path="$EMBED_DIR/face_rec.onnx"
  local zip_path="$EMBED_DIR/buffalo_sc.zip"
  local extract_dir="$EMBED_DIR/buffalo_sc"
  local url="https://github.com/deepinsight/insightface/releases/download/v0.7/buffalo_sc.zip"

  if [[ -f "$path" ]]; then
    echo "Face rec model exists: $path"
    return
  fi

  echo "Downloading InsightFace buffalo_sc..."
  curl -fsSL "$url" -o "$zip_path"
  rm -rf "$extract_dir"
  mkdir -p "$extract_dir"
  unzip -qo "$zip_path" -d "$extract_dir"

  if [[ -f "$extract_dir/w600k_mbf.onnx" ]]; then
    cp -f "$extract_dir/w600k_mbf.onnx" "$path"
  elif [[ -f "$extract_dir/buffalo_sc/w600k_r50.onnx" ]]; then
    cp -f "$extract_dir/buffalo_sc/w600k_r50.onnx" "$path"
  elif [[ -f "$extract_dir/buffalo_sc/w600k_mbf.onnx" ]]; then
    cp -f "$extract_dir/buffalo_sc/w600k_mbf.onnx" "$path"
  else
    echo "error: face_rec.onnx not found in buffalo_sc package" >&2
    exit 1
  fi

  rm -f "$zip_path"
  rm -rf "$extract_dir"
  echo "Saved: $path"
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

ensure_yolo_onnx
ensure_face_detect_onnx
ensure_face_rec_onnx

download_ort_dylib "arm64" \
  "https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/onnxruntime-osx-arm64-${ORT_VERSION}.tgz" \
  "$EMBED_DIR/darwin_arm64"

download_ort_dylib "x86_64" \
  "https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/onnxruntime-osx-x86_64-${ORT_VERSION}.tgz" \
  "$EMBED_DIR/darwin_amd64"

echo "Darwin embed assets ready in $EMBED_DIR/"
