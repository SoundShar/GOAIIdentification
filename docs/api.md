# API 文档

服务地址：`http://127.0.0.1:7986`

## GET /api/health

健康检查。

```json
{ "status": "ok", "version": "1.0.0" }
```

## POST /api/init

设置换人检测基准人脸（进程内存，重启失效）。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `master_face` | file | 是 | JPEG/PNG 基准人脸 |

成功：

```json
{ "ok": true, "message": "master face initialized" }
```

失败：400（未检测到人脸、格式错误等）

## POST /api/upload

上传图片并返回识别结果。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `image` | file | 是 | JPEG/PNG，最大 10MB |

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
