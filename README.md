# yks-tool 考试服务工具

纯 Go 实现的本地后台 HTTPS 服务，无 Web 界面，供本机 HTTPS 考试页通过域名直连调用。集成 YOLO11 ONNX 监考识别（对齐 aiIdentification 八种告警）。

启动后以系统托盘（Windows）或菜单栏图标（macOS）驻留后台，监听 `127.0.0.1:7986`（HTTPS）。对外访问地址：`https://local.cetset.com:7986`（需 DNS 将 `local.cetset.com` 解析到 `127.0.0.1`）。

## 功能特性

- **本地 HTTPS 服务**：监听 `127.0.0.1:7986`；本机生成 CA/叶子证并提权写入系统信任库
- **AI 图片识别**：`POST /api/upload` 返回无人/多人/换人/转头/低头/范围/书籍/手机
- **基准人脸**：`POST /api/init` 设置换人比对基准脸
- **健康检查**：`GET /api/health`
- **请求日志**：macOS `~/Library/Logs/yks-tool/app.log`；Windows `%LOCALAPPDATA%\yks-tool\logs\app.log`
- **系统托盘 / 菜单栏退出**
- **启动提示**：HTTPS 服务就绪后弹出原生对话框「运行考试服务成功」（Windows MessageBox / macOS 系统对话框）

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
chmod +x scripts/build-darwin.sh scripts/download-deps-darwin.sh scripts/sign-notarize-darwin.sh
./scripts/build-darwin.sh
# 产物：build/yks-tool.app（Universal，未签名；中间二进制构建后删除）
open build/yks-tool.app
```

正式分发（签名 + 公证；默认复用 it-ogt-pc-mac 的 Developer ID / notary profile）：

```bash
# 可选覆盖；不设则默认：
#   YKS_APPLE_IDENTITY='BBAB30F5901351F4F769DFEEF702BAF26CE968C4'  # 证书 SHA-1，避免同名 ambiguous
#   YKS_NOTARY_PROFILE='com.seaskylight.yksmacos'   # 与 Electron 考试端共用钥匙串 profile
./scripts/sign-notarize-darwin.sh
# 产物：build/yks-tool.app（stapled）+ build/yks-tool-macos.zip；临时公证包与 lipo 中间文件会删掉
```

`build-darwin.sh` 会调用 `download-deps-darwin.sh` 自动准备 `embeddata/*.onnx` 与各架构 `libonnxruntime.dylib`（YOLO 导出需本机 Python + `ultralytics`），再 `lipo` 合成 Universal 并组装 `.app`（Bundle ID：`com.seaskylight.ykstool`）。安装：解压 zip 后拖到「应用程序」或直接打开。

## 产物对照

| 平台 | 产物 | 内嵌原生库 |
|------|------|------------|
| Windows x64 | `build/yks-tool.exe` | `onnxruntime.dll` |
| macOS Universal | `build/yks-tool.app` / `build/yks-tool-macos.zip` | 各切片嵌入对应 `libonnxruntime.dylib` |

一套源码，**各平台单独编译**；不可用一个 exe 跨 Win/Mac。macOS 最低系统 12.0，菜单栏常驻（`LSUIElement`）。

## API 摘要

服务地址：`https://local.cetset.com:7986`（浏览器须用域名，不能用 `https://127.0.0.1:7986`，证书 SAN 不匹配）

**前置条件：**

- `local.cetset.com` A 记录或 hosts 指向 `127.0.0.1`
- 首次启动同意系统提权安装本机根证书（拒绝则无法启动 HTTPS；下次打开再提示）
- Firefox 默认不信任系统 CA，需手动导入本机 `ca.crt`，或使用 Safari / Chrome / Edge

### 设置基准人脸

```bash
curl -X POST https://local.cetset.com:7986/api/init -F "master_face=@face.jpg"
```

### 上传识别

```bash
curl -X POST https://local.cetset.com:7986/api/upload -F "image=@photo.jpg"
```

响应含 `detection`（8 项布尔）与 `codes`（行为码列表）。详见 [docs/api.md](docs/api.md)。

### 行为码

| code | 说明 | 模型 / 技术 |
|------|------|-------------|
| 1001 | 无人 | YOLO11（COCO `person` 计数 &lt; 1） |
| 1002 | 多人 | YOLO11（COCO `person` 计数 &gt; 1） |
| 1003 | 疑似手机 | YOLO11（`cell phone` / `remote`） |
| 1004 | 疑似书籍 | YOLO11（`book`） |
| 1005 | 疑似换人 | 单人且关键点在围栏内、无低头/转头、`face.score >= 0.7` 时：YuNet + InsightFace embedding 与 `/api/init` 基准比对（相似度 `< 0.4`） |
| 2001 | 低头 | YuNet 关键点 + 头部姿态（`pitch < -15`） |
| 2002 | 转头 | YuNet 关键点 + 头部姿态（`|yaw| > 30`；可与低头同时触发；不做 roll） |
| 2003 | 越界（四边 0.2 内框） | 五关键点相对画面四边各 0.2 边距的居中内框；越界只报此项，跳过低头/转头/换人 |

推理运行时：ONNX Runtime；模型内嵌于 exe（`yolo11.onnx` / `face_detect.onnx` / `face_rec.onnx`）。

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
├── assets_embed_tls.go            # publicServiceURL + loadTLSCertificate
├── tls_local_ca.go                # 本机 CA / 叶子证
├── tls_trust_*.go                 # 系统信任库提权安装
├── versioninfo.json     # Windows 文件版本（CompanyName=com.seaskylight.ykstool）
├── packaging/macos/     # Info.plist / entitlements / AppIcon.icns
├── embeddata/           # 构建用嵌入资源（download-deps 生成，打入二进制）
├── scripts/
│   ├── download-deps.ps1
│   ├── download-deps-darwin.sh
│   ├── build-windows.ps1
│   ├── build-darwin.sh
│   └── sign-notarize-darwin.sh
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
| 对外 URL | `https://local.cetset.com:7986` |
| 上传字段 | `image` |
| 最大上传 | 10MB |
| 模型 | 内嵌于 exe（可用 `YKS_MODEL_DIR` 覆盖为外挂目录） |

环境变量：`YKS_MODEL_DIR`、`YKS_ORT_DLL`、`YKS_ORT_LIB`、`YKS_SKIP_DETECTOR`、`AIWEB_CONSOLE`、`YKS_HTTP_ONLY`（`1` 时跳过本机 CA/提权，回退 HTTP，仅开发调试）、`YKS_CORS_ORIGIN`（可选，覆盖内置考试页 CORS 白名单）

## 依赖

- [github.com/getlantern/systray](https://github.com/getlantern/systray)
- [github.com/yalue/onnxruntime_go](https://github.com/yalue/onnxruntime_go)
- ONNX Runtime 动态库 + YOLO11 / YuNet / InsightFace 模型
