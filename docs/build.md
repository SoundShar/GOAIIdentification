# 构建与发布

## 环境要求

- Go 1.21+（推荐 1.26+）
- Windows：**MinGW-w64 / gcc**（CGO 编译 `onnxruntime_go`）
- PowerShell 5+
- Python 3 + `ultralytics`（构建时导出 YOLO11 ONNX opset 17）

## Windows 单文件打包

```powershell
cd D:\dev\aiWeb
.\scripts\build-windows.ps1
```

脚本会：

1. 下载 ONNX 模型与 `onnxruntime.dll` 到 `embeddata/`
2. 生成 Windows 版本资源（`versioninfo.json`）
3. 编译为 **单个** `build/yks-tool.exe`（模型与 DLL 已嵌入）

产物约 40MB，分发时只需 `yks-tool.exe` 一个文件。

依赖版本：`onnxruntime_go v1.12.1` + 内嵌 `onnxruntime 1.19.2` + YOLO11（opset 17）。

### 运行时

- ONNX 模型从 exe 内嵌字节加载，无需 `models/` 目录
- `onnxruntime.dll` 首次启动解压到 `%LOCALAPPDATA%\yks-tool\onnxruntime.dll`（版本一致时复用）

### Windows 文件版本

- 产品名称：`yks-tool`
- 文件说明：考试服务工具
- 版权：`com.seaskylight.yksmacos`
- 文件版本：`1.0.0.0`

## 开发调试

```powershell
.\scripts\download-deps.ps1
$env:AIWEB_CONSOLE = "1"
go run .
```

## 环境变量

| 变量 | 说明 |
|------|------|
| `YKS_MODEL_DIR` | 可选，指定外挂模型目录（调试用，覆盖内嵌模型） |
| `YKS_ORT_DLL` | 可选，指定 onnxruntime.dll 路径 |
| `YKS_SKIP_DETECTOR` | `1` 时跳过模型加载 |
| `AIWEB_CONSOLE` | `1` 时日志输出控制台 |
