# yks-tool 架构

## 概述

`yks-tool` 是纯 Go 实现的本地 HTTPS 托盘服务，监听 `127.0.0.1:7986`，对外 URL 为 `https://local.cetset.com:7986`，提供图片上传与 AI 监考识别能力。Windows 驻留系统托盘；macOS 以菜单栏 Agent（`LSUIElement`，不占 Dock）形态分发为 `yks-tool.app`。

HTTPS 考试页（如 `videoVIew.html`）通过域名跨端口调用本服务；服务端配置 CORS 白名单（默认 `https://local.cetset.com`）与 `Access-Control-Allow-Private-Network`，满足浏览器 Private Network Access 要求。

## TLS（本机 CA）

- 首次启动在用户目录生成 RSA-2048 本机 CA 与叶子证（SAN=`local.cetset.com`），不嵌入公网 DV 证书
  - Windows：`%LOCALAPPDATA%\yks-tool\ssl\`
  - macOS：`~/Library/Application Support/yks-tool/ssl/`
- 自动提权将根证写入系统信任库（Win：`certutil` + UAC；macOS：`security add-trusted-cert` + 管理员密码）
- **硬失败**：用户拒绝授权或安装失败 → 不监听 HTTPS，进程启动失败；下次打开再提权。不降级使用未信任证书
- 已信任且 CA 未变则不再弹提权；叶子证剩余 &lt;30 天用现有 CA 重签
- `YKS_HTTP_ONLY=1` 跳过 CA/提权，以 HTTP 调试
- DNS：`local.cetset.com` → `127.0.0.1`
- **Firefox** 使用独立证书库，默认不读系统信任；需手动导入本机 `ca.crt`，或改用 Chromium / Safari / Edge

## 模块

| 文件 | 职责 |
|------|------|
| `main.go` | 入口：日志、检测器异步初始化；**CA/提权+Listen 成功后再弹启动提示并进托盘** |
| `server.go` | 路由与 HTTPS 服务生命周期 |
| `handler.go` | `/api/health`、`/api/init`、`/api/upload` |
| `detector.go` | **单文件** YOLO11 + 人脸 ONNX 推理与 8 项告警 |
| `notice.go` / `notice_ui*.go` | 跨平台启动提示：TLS/Listen 结束后再弹窗（成功/失败）；Win/Mac 共用同一 HTML 页 |
| `assets_embed_tls.go` | `publicServiceURL` + `loadTLSCertificate()` 入口 |
| `tls_local_ca.go` | 本机 CA/叶子证目录、生成与复用 |
| `tls_trust_*.go` | 系统信任库检测与提权安装（硬失败） |
| `middleware.go` | 请求日志、CORS、Private Network |
| `tray.go` | 系统托盘 |
| `logger.go` | slog 文件日志（macOS：`~/Library/Logs/yks-tool/`；Windows：`%LOCALAPPDATA%\yks-tool\logs\`） |

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
- **启动策略**：本机 CA/提权与 HTTPS Listen 成功后再进托盘；拒绝提权则进程退出、不驻留托盘。ONNX 会话在后台加载。`/api/init`、`/api/upload` 在未就绪时返回 `503`；`/api/health` 含 `detector` 字段（`ready` / `loading` / `error` / `skipped`）
- **启动提示**：主进程先完成 CA/提权与 HTTPS Listen，再拉起 `--notice-ui` 子进程（避免 macOS 上提示窗与管理员授权抢交互导致 `SecTrustSettings` 失败）；子进程轮询本机 `/api/health`，并读取 boot status（`ready`/`failed`）
- macOS 正式包 entitlements：`disable-library-validation`（加载非本 Team 的 ORT dylib）、网络 client/server
- 构建需 **CGO**（Windows：MinGW；macOS：clang）

## 参考

检测阈值与行为码对齐 [aiIdentification](d:\dev\aiIdentification) `src/yolo/meta.ts`。
