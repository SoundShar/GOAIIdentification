# 约定

## 命名

- 产物：`yks-tool.exe`（Windows）、`yks-tool`（macOS）
- 产品名 / 托盘：`yks-tool`
- 检测 JSON 键与 aiIdentification `meta.ts` 告警键一致

## 识别

- 单帧检测，无防抖（`changeTest` 帧累计逻辑不在本服务实现）
- YOLO 无人/多人时跳过人脸管线
- 围栏：画面宽高各 80% 居中矩形

## 构建

- 必须 `CGO_ENABLED=1` 与 gcc（Windows）
- 打包前执行 `download-deps.ps1` 与 `goversioninfo`
