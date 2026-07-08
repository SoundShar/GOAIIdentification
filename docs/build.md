# 构建与发布

## 环境要求

- Go 1.21+（推荐 1.26+）
- Python 3 + `ultralytics`（构建时导出 YOLO11 ONNX opset 17，Windows 脚本或 Mac 手动导出）

依赖版本：`onnxruntime_go v1.12.1` + ONNX Runtime **1.19.2** + YOLO11（opset 17）。

## 产物对照

| 平台 | 命令 | 产物 |
|------|------|------|
| Windows x64 | `.\scripts\build-windows.ps1` | `build/yks-tool.exe` |
| macOS arm64 | `./scripts/build-darwin.sh` | `build/yks-tool-darwin-arm64` |
| macOS amd64 | `./scripts/build-darwin.sh` | `build/yks-tool-darwin-amd64` |

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
- 版权：`com.seaskylight.yksmacos`

---

## macOS 单文件打包

### 环境

- **须在 macOS 上编译**（systray + CGO）
- Xcode Command Line Tools
- `embeddata/yolo11.onnx` 等三个模型（可自 Windows 拷贝）

### 命令

```bash
cd /path/to/aiWeb
chmod +x scripts/build-darwin.sh scripts/download-deps-darwin.sh
./scripts/build-darwin.sh
```

脚本会：

1. `download-deps-darwin.sh` 下载：
   - `embeddata/darwin_arm64/libonnxruntime.dylib`
   - `embeddata/darwin_amd64/libonnxruntime.dylib`
2. 分架构编译（`GOOS=darwin` 相同，`GOARCH` 不同）：
   - `arm64` → `yks-tool-darwin-arm64`（嵌入 ARM64 dylib）
   - `amd64` → `yks-tool-darwin-amd64`（嵌入 x86_64 dylib）

在 Apple Silicon 上交叉编 Intel 包时，脚本自动使用 `CC="clang -arch x86_64"`。

### 运行时

- `libonnxruntime.dylib` 首次解压到 `~/Library/Caches/yks-tool/`

### 架构说明

| 机器 | GOOS | GOARCH | 内嵌库目录 |
|------|------|--------|------------|
| Apple Silicon | darwin | arm64 | `embeddata/darwin_arm64/` |
| Intel Mac | darwin | amd64 | `embeddata/darwin_amd64/` |

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
./scripts/download-deps-darwin.sh   # 需已有 embeddata/*.onnx
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
