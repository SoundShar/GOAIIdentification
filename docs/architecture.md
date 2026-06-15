# yks-tool 架构

## 概述

`yks-tool` 是纯 Go 实现的本地 HTTP 托盘服务，监听 `127.0.0.1:7986`，提供图片上传与 AI 监考识别能力。

## 模块

| 文件 | 职责 |
|------|------|
| `main.go` | 入口：日志、检测器初始化、HTTP、托盘 |
| `server.go` | 路由与 HTTP 服务生命周期 |
| `handler.go` | `/api/health`、`/api/init`、`/api/upload` |
| `detector.go` | **单文件** YOLO11 + 人脸 ONNX 推理与 8 项告警 |
| `middleware.go` | 请求日志、CORS |
| `tray.go` | 系统托盘 |
| `logger.go` | slog 文件日志 |

## 识别链路

```text
POST /api/upload (image)
  → JPEG/PNG 解码
  → detector.AnalyzeImage
       ├─ YOLO11：person 计数、book、cell phone/remote
       └─ 单人时 YuNet：低头/转头/越界
            └─ w600k_mbf：与 /api/init 基准 embedding 比对（换人）
  → JSON detection + codes
```

基准人脸：`POST /api/init` 上传 `master_face`，embedding 存于进程内存，重启后需重新设置。

## 模型与运行时

- 推理：`github.com/yalue/onnxruntime_go v1.12.1` + ONNX Runtime 1.19.2
- **单文件分发**：YOLO / 人脸 ONNX 与平台原生库在构建时嵌入（`go:embed`）
  - Windows：`assets_embed_windows.go` → `onnxruntime.dll`
  - macOS arm64：`assets_embed_darwin_arm64.go` → `darwin_arm64/libonnxruntime.dylib`
  - macOS amd64：`assets_embed_darwin_amd64.go` → `darwin_amd64/libonnxruntime.dylib`
  - 共用：`assets_embed_common.go` → 三个 `.onnx`
- 运行时原生库解压至用户缓存 `yks-tool/`；模型从内存加载
- 构建需 **CGO**（Windows：MinGW；macOS：clang）

## 参考

检测阈值与行为码对齐 [aiIdentification](d:\dev\aiIdentification) `src/yolo/meta.ts`。
