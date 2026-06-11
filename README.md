# aiWeb 本地 HTTP 服务

纯 Go 实现的本地后台 HTTP 服务，无 Web 界面，供本机其他程序（浏览器、桌面应用、脚本等）通过 HTTP 调用。

启动后以系统托盘（Windows）或菜单栏图标（macOS）驻留后台，监听 `127.0.0.1:8080`，支持图片上传校验与全链路请求日志。

## 功能特性

- **本地 HTTP 服务**：仅监听 `127.0.0.1`，仅本机可访问
- **图片上传接口**：`multipart/form-data` 二进制上传，内存校验 JPEG/PNG，不落盘
- **健康检查**：`/api/health` 探测服务存活
- **请求日志**：每个接口调用写入 `logs/app.log`
- **系统托盘退出**：Windows 托盘 / macOS 菜单栏 →「退出」，优雅关闭 HTTP 服务
- **CORS**：默认开启，支持浏览器跨域调用

## 环境要求

- Go 1.21+（推荐 1.26+）
- Windows：无需额外运行时
- macOS：建议在 macOS 本机编译（systray 依赖系统 GUI 框架）

## 项目结构

```text
aiWeb/
├── main.go              # 入口：托盘 + HTTP + 优雅退出
├── server.go            # HTTP 服务与路由
├── handler.go           # 接口业务逻辑
├── middleware.go        # 日志与 CORS 中间件
├── tray.go              # 托盘 / 菜单栏
├── logger.go            # slog 日志初始化
├── assets/              # 托盘图标（icon.ico / icon.png）
├── logs/                # 运行时日志目录（自动创建）
├── scripts/
│   ├── build-windows.ps1
│   └── build-darwin.sh
└── build/               # 打包产物目录
```

## 快速开始

### 开发模式

```powershell
cd D:\dev\aiWeb

# 可选：同时输出日志到控制台
$env:AIWEB_CONSOLE = "1"

go run .
```

启动后任务栏托盘会出现图标（可能在「隐藏图标」区域）。

### 打包

**Windows：**

```powershell
.\scripts\build-windows.ps1
# 产物：build\myapp.exe
```

**macOS（需在 Mac 上执行）：**

```bash
chmod +x scripts/build-darwin.sh
./scripts/build-darwin.sh
# 产物：build/myapp
```

双击 `myapp.exe` 运行，无控制台窗口，仅显示托盘图标。

### 退出服务

- 托盘 / 菜单栏图标 → 点击「退出」
- 或在任务管理器中结束 `myapp.exe` 进程

## API 说明

服务地址：`http://127.0.0.1:8080`

### 健康检查

```
GET /api/health
```

响应示例：

```json
{
  "status": "ok",
  "version": "1.0.0"
}
```

### 图片上传

```
POST /api/upload
Content-Type: multipart/form-data
```

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `image` | file | 是 | 图片文件，支持 JPEG、PNG |

限制：

- 单文件最大 **10MB**
- 仅校验格式与尺寸，**不写入磁盘**

成功响应示例：

```json
{
  "ok": true,
  "filename": "photo.jpg",
  "contentType": "image/jpeg",
  "size": 102400,
  "width": 1920,
  "height": 1080
}
```

错误码：

| 状态码 | 说明 |
|---|---|
| 400 | 缺少字段、图片格式无效 |
| 413 | 文件超过大小限制 |
| 405 | 请求方法不正确 |
| 500 | 服务内部错误 |

## 调用示例

### curl

```bash
# 健康检查
curl http://127.0.0.1:8080/api/health

# 上传图片
curl -X POST http://127.0.0.1:8080/api/upload -F "image=@photo.jpg"
```

### JavaScript（浏览器）

```javascript
const formData = new FormData()
formData.append('image', fileInput.files[0])

const res = await fetch('http://127.0.0.1:8080/api/upload', {
  method: 'POST',
  body: formData
})
const data = await res.json()
console.log(data)
```

## 日志

日志文件：`logs/app.log`（自动创建、追加写入）

每个 HTTP 请求记录：

- `method`、`path`、`status`
- `duration_ms`（耗时）
- `bytes_in`、`bytes_out`（传输大小）
- `remote`（调用方地址）
- `user_agent`

图片上传成功时额外记录 `upload_validated`：

```text
level=INFO msg=upload_validated filename=photo.jpg content_type=image/jpeg format=jpeg width=1920 height=1080 size=102400
```

开发调试时可设置环境变量 `AIWEB_CONSOLE=1`，日志同时输出到控制台。

## 配置说明

当前版本使用代码内常量，主要配置如下：

| 配置项 | 默认值 | 位置 |
|---|---|---|
| 监听地址 | `127.0.0.1:8080` | `server.go` |
| 上传字段名 | `image` | `handler.go` |
| 最大上传大小 | 10MB | `handler.go` |
| 日志文件 | `logs/app.log` | `logger.go` |

## 注意事项

1. **仅本机访问**：服务绑定 `127.0.0.1`，局域网其他设备无法直接访问；如需对外暴露需修改监听地址并增加鉴权。
2. **图片不落盘**：上传图片仅在内存中校验，请求结束后由 GC 回收。
3. **Mac 构建**：带 systray 的程序建议在 macOS 本机编译，Windows 交叉编译可能受限。
4. **Mac 签名**：当前未做代码签名与公证，分发给他人在 macOS 上可能提示「无法验证开发者」。
5. **托盘图标**：可替换 `assets/icon.ico`（Windows）和 `assets/icon.png`（macOS）后重新打包。

## 依赖

- [github.com/getlantern/systray](https://github.com/getlantern/systray) — 系统托盘 / 菜单栏

其余均为 Go 标准库（`net/http`、`log/slog`、`image` 等）。

## License

内部项目，按需补充许可证说明。
