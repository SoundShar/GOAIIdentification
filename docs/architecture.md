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
| `detector.go` / `detector_env.go` | YOLO11 + 人脸 ONNX 推理与 8 项告警；阈值「常量默认 + 环境变量覆盖」 |
| `notice.go` / `notice_*.go` | 跨平台启动提示：HTTPS Listen 成功后弹原生对话框（成功/失败） |
| `assets_embed_tls.go` | `publicServiceURL` + `loadTLSCertificate()` 入口 |
| `tls_local_ca.go` | 本机 CA/叶子证目录、生成与复用 |
| `tls_trust_*.go` | 系统信任库检测与提权安装（硬失败） |
| `middleware.go` | CORS、Private Network |
| `tray.go` | 系统托盘 |
| `logger.go` | slog 文件日志（macOS：`~/Library/Logs/yks-tool/`；Windows：`%LOCALAPPDATA%\yks-tool\logs\`） |

## 识别链路

```text
POST /api/upload (image)
  → JPEG/PNG 解码
  → detector.AnalyzeImage
       ├─ YOLO11：person 计数 + person 框、book、cell phone/remote
       ├─ 无人 / 多人：置对应标志，不进人脸管线
       └─ 单人：
            ├─ 五关键点相对四边 0.2 内框：任一点出界 → 仅 rangeTestPC（不跑低头/转头/换人）
            └─ 关键点在围栏内 → 低头/转头 +（质量门后）w600k_mbf 换人比对
  → JSON detection + codes
```

姿态阈值（`detector.go`）：低头 `pitch < -15`；转头 `|yaw| > 30`（独立判定，可同时触发；不做 roll）。越界认五关键点 × 四边 0.2 内框（不用 YOLO 全身框）。换人仅在单人、关键点在围栏内、无低头/转头且 `face.score >= 0.7` 时比对，相似度 `< 0.4` 报换人。

基准人脸：`POST /api/init` 上传 `master_face`，embedding 存于进程内存，重启后需重新设置。

## 检测阈值

行为码与告警字段对齐 [aiIdentification](d:\dev\aiIdentification) `src/yolo/meta.ts`。阈值采用**常量默认值 + 可选环境变量覆盖**（见 `docs/build.md`），未设置环境变量时与历史硬编码行为一致（零回归结构修复）：

| 判定 | 默认 | 说明 |
|------|------|------|
| YOLO person / phone / remote / book | `0.2` | 各类别可独立覆盖；人数仍为 `personCount<1` 无人、`>1` 多人（不拆两档阈值） |
| 电子围栏容差 | `0.03` | 相对围栏宽高外扩后再比脸框四角 |
| 低头 / 转头 | pitch`<-9`；yaw/roll/pitch 上限同历史 | 已低头或转头时跳过换人比对 |
| 换人相似度 | `0.4` | 余弦相似度低于阈值判换人 |

Debug 埋点：`yolo_scores`（仅 person / cell phone / remote / book 的原始分）、`head_pose`（pitch/yaw/roll）；`AIWEB_CONSOLE=1` 时日志级别为 Debug 可采集。

## 模型与运行时

- 推理：`github.com/yalue/onnxruntime_go v1.12.1` + ONNX Runtime 1.19.2
- **单文件分发**：YOLO / 人脸 ONNX 与平台原生库在构建时嵌入（`go:embed`）
  - Windows：`assets_embed_windows.go` → `onnxruntime.dll`
  - macOS arm64：`assets_embed_darwin_arm64.go` → `darwin_arm64/libonnxruntime.dylib`
  - macOS amd64：`assets_embed_darwin_amd64.go` → `darwin_amd64/libonnxruntime.dylib`
  - 共用：`assets_embed_common.go` → 三个 `.onnx`
- 运行时原生库解压至用户缓存 `yks-tool/`；模型从内存加载
- **启动策略**：本机 CA/提权与 HTTPS Listen 成功后再进托盘；拒绝提权则进程退出、不驻留托盘。ONNX 会话在后台加载。`/api/init`、`/api/upload` 在未就绪时返回 `503`；`/api/health` 含 `detector` 字段（`ready` / `loading` / `error` / `skipped`）
- **启动提示**：HTTPS Listen 完成后在主进程弹原生对话框；成功「运行考试服务成功」，失败「启动考试服务失败」后进程退出
- macOS 正式包 entitlements：`disable-library-validation`（加载非本 Team 的 ORT dylib）、网络 client/server
- 构建需 **CGO**（Windows：MinGW；macOS：clang）

## 参考

行为码与前端告警键对齐 [aiIdentification](d:\dev\aiIdentification) `src/yolo/meta.ts`；具体阈值数值以本仓库默认值与环境变量为准，不强制对齐外部 Python 项目。
