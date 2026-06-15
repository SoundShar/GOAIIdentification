# yks-tool 考试服务工具

纯 Go 实现的本地后台 HTTP 服务，无 Web 界面，供本机其他程序通过 HTTP 调用。集成 YOLO11 ONNX 监考识别（对齐 aiIdentification 八种告警）。

启动后以系统托盘（Windows）或菜单栏图标（macOS）驻留后台，监听 `127.0.0.1:7986`。

## 功能特性

- **本地 HTTP 服务**：仅监听 `127.0.0.1`
- **AI 图片识别**：`POST /api/upload` 返回无人/多人/换人/转头/低头/范围/书籍/手机
- **基准人脸**：`POST /api/init` 设置换人比对基准脸
- **健康检查**：`GET /api/health`
- **请求日志**：`logs/app.log`
- **系统托盘退出**

## 环境要求

- Go 1.21+（推荐 1.26+）
- Windows：MinGW/gcc（CGO）+ `onnxruntime.dll`
- 首次使用需下载 ONNX 模型（见下方）

## 快速开始

### 1. 打包（单文件，模型已内嵌）

```powershell
cd D:\dev\aiWeb
.\scripts\build-windows.ps1
# 产物：build\yks-tool.exe（独立可执行，无需外挂 models/dll）
```

脚本会自动下载模型到 `embeddata/` 并编译进 exe。

### 2. 开发运行

```powershell
.\scripts\download-deps.ps1   # 首次需下载 embeddata/
$env:AIWEB_CONSOLE = "1"
go run .
```

双击 `yks-tool.exe` 即可运行，无需额外文件。首次启动会将内嵌的 ONNX Runtime 解压到用户缓存目录。

### 3. macOS 打包（须在 Mac 上执行）

```bash
chmod +x scripts/build-darwin.sh scripts/download-deps-darwin.sh
./scripts/build-darwin.sh
# 产物：
#   build/yks-tool-darwin-arm64  （Apple Silicon）
#   build/yks-tool-darwin-amd64  （Intel Mac）
```

需先有 `embeddata/*.onnx`（可在 Windows 跑 `download-deps.ps1` 后拷贝，或在 Mac 上导出）。脚本会下载各架构 `libonnxruntime.dylib` 并分别打入对应二进制。

## 产物对照

| 平台 | 产物 | 内嵌原生库 |
|------|------|------------|
| Windows x64 | `build/yks-tool.exe` | `onnxruntime.dll` |
| macOS arm64 | `build/yks-tool-darwin-arm64` | ARM64 `libonnxruntime.dylib` |
| macOS amd64 | `build/yks-tool-darwin-amd64` | x86_64 `libonnxruntime.dylib` |

一套源码，**各平台单独编译**；不可用一个 exe 跨 Win/Mac。

## API 摘要

服务地址：`http://127.0.0.1:7986`

### 设置基准人脸

```bash
curl -X POST http://127.0.0.1:7986/api/init -F "master_face=@face.jpg"
```

### 上传识别

```bash
curl -X POST http://127.0.0.1:7986/api/upload -F "image=@photo.jpg"
```

响应含 `detection`（8 项布尔）与 `codes`（行为码列表）。详见 [docs/api.md](docs/api.md)。

### 行为码

| code | 说明 |
|------|------|
| 1001 | 无人 |
| 1002 | 多人 |
| 1003 | 疑似手机 |
| 1004 | 疑似书籍 |
| 1005 | 疑似换人 |
| 2001 | 低头 |
| 2002 | 转头 |
| 2003 | 越界（80% 居中框） |

## 项目结构

```text
aiWeb/
├── main.go
├── server.go
├── handler.go
├── detector.go          # AI 识别（单文件）
├── assets_embed_common.go
├── assets_embed_windows.go
├── assets_embed_darwin_arm64.go   # build tag: darwin && arm64
├── assets_embed_darwin_amd64.go   # build tag: darwin && amd64
├── versioninfo.json     # Windows 文件版本
├── embeddata/           # 构建用嵌入资源（download-deps 生成，打入二进制）
├── scripts/
│   ├── download-deps.ps1
│   ├── download-deps-darwin.sh
│   ├── build-windows.ps1
│   └── build-darwin.sh
├── docs/
└── build/               # 打包产物
```

## 文档

- [架构](docs/architecture.md)
- [API](docs/api.md)
- [构建发布](docs/build.md)

## 配置

| 配置项 | 默认值 |
|--------|--------|
| 监听地址 | `127.0.0.1:7986` |
| 上传字段 | `image` |
| 最大上传 | 10MB |
| 模型 | 内嵌于 exe（可用 `YKS_MODEL_DIR` 覆盖为外挂目录） |

环境变量：`YKS_MODEL_DIR`、`YKS_ORT_DLL`、`YKS_ORT_LIB`、`YKS_SKIP_DETECTOR`、`AIWEB_CONSOLE`

## 依赖

- [github.com/getlantern/systray](https://github.com/getlantern/systray)
- [github.com/yalue/onnxruntime_go](https://github.com/yalue/onnxruntime_go)
- ONNX Runtime 动态库 + YOLO11 / YuNet / InsightFace 模型
