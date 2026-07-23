# API 文档

服务地址：`https://local.cetset.com:7986`

**前置条件：**

- `local.cetset.com` 须解析到 `127.0.0.1`（hosts 或 DNS）
- 浏览器访问须使用域名（证书 SAN 为 `local.cetset.com`）
- 首次启动须同意提权安装本机根证书；拒绝或失败则 HTTPS 不启动，下次打开再提权
- Firefox 使用独立证书库，默认不读系统信任，需手动导入本机 `ca.crt`（Windows：`%LOCALAPPDATA%\yks-tool\ssl\ca.crt`；macOS：`~/Library/Application Support/yks-tool/ssl/ca.crt`），或改用系统浏览器
- HTTPS 考试页跨端口调用本服务时，服务端返回 CORS 白名单 Origin 与 `Access-Control-Allow-Private-Network: true`
- 内置允许的考试页 Origin：`https://yk.cetset.com`、`https://test.cetset.com`、`https://kspre.yks365.net`、`https://ks.yks365.net`、`https://local.cetset.com`（可用 `YKS_CORS_ORIGIN` 覆盖）

系统已信任本机 CA 后，curl 无需 `-k`：

```bash
curl https://local.cetset.com:7986/api/health
```

## GET /api/health

健康检查。`detector` 表示识别引擎状态：`ready` / `loading` / `error` / `skipped`。

```json
{ "status": "ok", "version": "1.0.0", "detector": "ready" }
```

服务启动后 HTTPS 会立即可用，模型在后台加载；`detector` 为 `loading` 时请稍后重试识别接口。

## POST /api/init

设置换人检测基准人脸（进程内存，重启失效）。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `master_face` | file | 是 | JPEG/PNG 基准人脸 |

成功：

```json
{ "ok": true, "message": "master face initialized" }
```

失败：400（未检测到人脸、格式错误等）；503（模型仍在加载或初始化失败）

## POST /api/upload

上传图片并返回识别结果。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `image` | file | 是 | JPEG/PNG，最大 10MB |

模型未就绪时返回 503（勿把空 detection 当作正常结果）。

成功示例：

```json
{
  "ok": true,
  "filename": "photo.jpg",
  "contentType": "image/jpeg",
  "size": 102400,
  "width": 1920,
  "height": 1080,
  "detection": {
    "nobodyPC": false,
    "multiplePersonPC": false,
    "findPhonePC": false,
    "findBookPC": false,
    "lowerHeadPC": false,
    "turnheadPC": false,
    "rangeTestPC": false,
    "changePersonPC": false
  },
  "codes": []
}
```

### 行为码

| 键 | code | 说明 |
|----|------|------|
| `nobodyPC` | 1001 | 无人 |
| `multiplePersonPC` | 1002 | 多人 |
| `findPhonePC` | 1003 | 疑似手机 |
| `findBookPC` | 1004 | 疑似书籍 |
| `changePersonPC` | 1005 | 疑似换人 |
| `lowerHeadPC` | 2001 | 低头 |
| `turnheadPC` | 2002 | 转头 |
| `rangeTestPC` | 2003 | 人像不在检测框内（80% 居中区域） |

`codes` 为当前帧命中码列表。
