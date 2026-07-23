# 嵌入资源目录

构建前准备 ONNX 模型与 ONNX Runtime 原生库，编译时打入 `yks-tool` 单文件。

## 共用模型（三端相同）

- `yolo11.onnx`（opset 17）
- `face_detect.onnx`
- `face_rec.onnx`

Windows：`scripts/download-deps.ps1`  
macOS：`scripts/download-deps-darwin.sh`（自动导出 YOLO + 下载人脸模型；需 Python/`ultralytics`）

## 平台原生库

| 平台 | 路径 | 获取脚本 |
|------|------|----------|
| Windows | `onnxruntime.dll` | `download-deps.ps1` |
| macOS arm64 | `darwin_arm64/libonnxruntime.dylib` | `download-deps-darwin.sh` |
| macOS amd64 | `darwin_amd64/libonnxruntime.dylib` | `download-deps-darwin.sh` |

大文件已 gitignore，不在仓库中。
