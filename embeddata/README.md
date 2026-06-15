# 嵌入资源目录

构建前执行 `scripts/download-deps.ps1`，会将 ONNX 模型与 `onnxruntime.dll` 下载到此目录并打入 `yks-tool.exe`。

所需文件：

- `yolo11.onnx`
- `face_detect.onnx`
- `face_rec.onnx`
- `onnxruntime.dll`

此目录下大文件已 gitignore，不在仓库中。
