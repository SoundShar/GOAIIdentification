# 构建与发布

## 环境要求

- Go 1.21+（推荐 1.26+）
- Python 3 + `ultralytics`（构建时导出 YOLO11 ONNX opset 17，Windows 脚本或 Mac 手动导出）

依赖版本：`onnxruntime_go v1.12.1` + ONNX Runtime **1.19.2** + YOLO11（opset 17）。

## 产物对照

| 平台 | 命令 | 产物 |
|------|------|------|
| Windows x64 | `.\scripts\build-windows.ps1` | `build/yks-tool.exe` |
| macOS（开发） | `./scripts/build-darwin.sh` | `build/yks-tool.app`（Universal，未签名）+ 中间二进制 |
| macOS（正式） | 上一步 + `./scripts/sign-notarize-darwin.sh` | 公证 staple 后的 `.app` + `build/yks-tool-macos.zip` |

一套源码，各平台**单独编译**；模型 ONNX 三端共用，原生库按平台/架构嵌入。

---

## Windows 单文件打包

### 环境

- MinGW-w64 / gcc（CGO）
- PowerShell 5+

### 命令

```powershell
cd D:\dev\aiWeb
.\scripts\build-windows.ps1
```

脚本会：

1. 下载/导出资源到 `embeddata/`（含 `onnxruntime.dll`）
2. 校验 `ssl/local.sharas.cn_bundle.crt` 与 `.key`，复制到 `embeddata/ssl/`
3. 生成 Windows 版本资源（`versioninfo.json`）
4. 编译 `build/yks-tool.exe`（模型、DLL、TLS 证书已嵌入）

产物约 40MB，分发只需 `yks-tool.exe`。

### 运行时

- ONNX 模型从内存加载
- `onnxruntime.dll` 首次解压到 `%LOCALAPPDATA%\yks-tool\`
- 默认以 **HTTPS** 监听 `127.0.0.1:7986`，对外 URL：`https://local.sharas.cn:7986`
- 日志：`%LOCALAPPDATA%\yks-tool\logs\app.log`

### TLS 与 DNS

| 项 | 说明 |
|----|------|
| 证书源 | `ssl/local.sharas.cn_bundle.crt` + `ssl/local.sharas.cn.key` |
| 构建嵌入 | 复制到 `embeddata/ssl/` 后打入 exe |
| DNS | `local.sharas.cn` → `127.0.0.1` |
| 开发回退 | `YKS_HTTP_ONLY=1` 时使用 HTTP（仅本机调试） |

### 联调验收

```powershell
# 1. hosts 或 DNS：local.sharas.cn -> 127.0.0.1
# 2. 启动 yks-tool.exe
curl -k https://local.sharas.cn:7986/api/health
# 预期：{"status":"ok",...}

# 3. 浏览器打开 https://local.sharas.cn/.../videoVIew.html
# 预期：无 Mixed Content / CORS 错误，控制台有上传日志
```

### 文件版本

- 产品名称：`yks-tool`
- 文件说明：考试服务工具
- 版权 / CompanyName：`com.seaskylight.ykstool`

---

## macOS .app 打包

### Bundle 约定

| 项 | 值 |
|----|-----|
| Bundle ID | `com.seaskylight.ykstool` |
| App 名 | `yks-tool.app` |
| 可执行文件 | `Contents/MacOS/yks-tool` |
| 显示名 | `考试服务工具` |
| 最低系统 | macOS 12.0（`LSMinimumSystemVersion` / `MACOSX_DEPLOYMENT_TARGET=12.0`） |
| UI | `LSUIElement=true`（菜单栏常驻，不占 Dock） |
| 架构 | Universal（`lipo`：arm64 + x86_64） |

模板与资源：`packaging/macos/`（`Info.plist`、`yks-tool.entitlements`、`AppIcon.icns`）。

### 环境

- **须在 macOS 上编译**（systray + CGO）
- Xcode Command Line Tools（`clang`、`lipo`、`sips`、`iconutil`；正式包另需 `codesign` / `notarytool`）
- Python 3 + 项目 `.venv`（推荐：`python3 -m venv .venv && .venv/bin/pip install ultralytics onnx`；首次导出 `yolo11.onnx`；人脸模型由脚本直接下载）

### 开发包（未签名）

```bash
cd /path/to/aiWeb
chmod +x scripts/build-darwin.sh scripts/download-deps-darwin.sh
./scripts/build-darwin.sh
# 可选：YKS_VERSION=1.0.1 ./scripts/build-darwin.sh
open build/yks-tool.app
```

脚本会：

1. `download-deps-darwin.sh` 导出/下载三个 ONNX 模型，并下载各架构 `libonnxruntime.dylib`
2. 校验并复制 `ssl/` 证书到 `embeddata/ssl/`（与 Windows 一致）
3. `MACOSX_DEPLOYMENT_TARGET=12.0` 下分别编译 arm64 / amd64
4. `lipo` 合成 Universal 二进制
5. 从 `assets/icon.png` 生成/刷新 `AppIcon.icns`
6. 组装 `build/yks-tool.app`

中间产物（便于排查）一并保留：

- `build/yks-tool-darwin-arm64`
- `build/yks-tool-darwin-amd64`
- `build/yks-tool-darwin-universal`

未签名 `.app` 本机联调时，若 Gatekeeper 拦截，可右键「打开」或清除隔离属性。

### 正式包（签名 + 公证 + staple）

默认与 `it-ogt-pc-mac`（`forge.js`）共用同一套凭据，无需再 export：

| 变量 | 默认值 |
|------|--------|
| `YKS_APPLE_IDENTITY` | `BBAB30F5901351F4F769DFEEF702BAF26CE968C4`（Developer ID Application SHA-1；与 Electron 考试端同一张证，有效期至 2031；用哈希避免钥匙串同名 ambiguous） |
| `YKS_NOTARY_PROFILE` | `com.seaskylight.yksmacos`（钥匙串 profile，可与 Electron 考试端共用） |

```bash
./scripts/build-darwin.sh
./scripts/sign-notarize-darwin.sh
# 分发：build/yks-tool-macos.zip（内含 stapled yks-tool.app）
```

也可临时覆盖，或改用 `YKS_APPLE_ID` + `YKS_APPLE_TEAM_ID` + `YKS_APPLE_APP_PASSWORD`。

签名脚本流程：`codesign`（Hardened Runtime + entitlements）→ `ditto` 提交 zip → `notarytool submit --wait` → `stapler staple` → 分发 `yks-tool-macos.zip`。

`yks-tool.entitlements` 含 `disable-library-validation` / `allow-unsigned-executable-memory`：Hardened Runtime 下需加载解压到 `~/Library/Caches/yks-tool/` 的第三方 `libonnxruntime.dylib`（与主程序 Team ID 不同）。与 it-ogt-pc-mac Electron 侧 entitlements 策略一致。

凭据默认写在脚本内（与 Electron 项目对齐），仍可用环境变量覆盖；缺身份或公证方式时脚本会失败退出。

### 运行时

- `libonnxruntime.dylib` 首次解压到 `~/Library/Caches/yks-tool/`
- 日志：`~/Library/Logs/yks-tool/app.log`

### 架构说明

| 机器 | GOOS | GOARCH | 内嵌库目录 |
|------|------|--------|------------|
| Apple Silicon | darwin | arm64 | `embeddata/darwin_arm64/` |
| Intel Mac | darwin | amd64 | `embeddata/darwin_amd64/` |

Universal `.app` 内嵌对应架构的原生库分别打入各切片二进制。

---

## 开发调试

**Windows：**

```powershell
.\scripts\download-deps.ps1
$env:AIWEB_CONSOLE = "1"
go run .
```

**macOS：**

```bash
./scripts/download-deps-darwin.sh   # 首次会准备 embeddata/*.onnx 与 dylib
export AIWEB_CONSOLE=1
go run .
```

## 环境变量

| 变量 | 说明 |
|------|------|
| `YKS_MODEL_DIR` | 可选，外挂模型目录（覆盖内嵌 ONNX） |
| `YKS_ORT_DLL` | 可选，指定 ONNX Runtime 库路径（Windows dll / Mac dylib） |
| `YKS_ORT_LIB` | 同 `YKS_ORT_DLL`（Mac 推荐别名） |
| `YKS_SKIP_DETECTOR` | `1` 时跳过模型加载 |
| `YKS_HTTP_ONLY` | `1` 时以 HTTP 启动（开发调试，默认 HTTPS） |
| `YKS_CORS_ORIGIN` | 可选，覆盖内置 CORS 白名单（逗号分隔）；默认已内置考试页域名 |
| `AIWEB_CONSOLE` | `1` 时日志输出控制台 |
| `YKS_VERSION` | macOS 构建时写入 `CFBundleShortVersionString` / `CFBundleVersion`，默认 `1.0.0` |
| `YKS_APPLE_IDENTITY` | 正式签名身份；默认证书 SHA-1 `BBAB30F5…`（SeaSkyLight `BVU65MZFLK`，与 it-ogt-pc-mac 相同） |
| `YKS_NOTARY_PROFILE` | `notarytool` keychain profile；默认 `com.seaskylight.yksmacos`（与 Electron 考试端共用） |
| `YKS_APPLE_ID` / `YKS_APPLE_TEAM_ID` / `YKS_APPLE_APP_PASSWORD` | 无 profile 时的公证凭据 |
| `YKS_APP_BUNDLE` | 可选，签名脚本目标 `.app`，默认 `build/yks-tool.app` |
