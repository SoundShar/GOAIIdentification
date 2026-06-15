# 约定

## 命名

- 产物：`yks-tool.exe`（Windows）、`yks-tool-darwin-arm64` / `yks-tool-darwin-amd64`（macOS）
- 产品名 / 托盘：`yks-tool`
- 检测 JSON 键与 aiIdentification `meta.ts` 告警键一致

## 识别

- 单帧检测，无防抖（`changeTest` 帧累计逻辑不在本服务实现）
- YOLO 无人/多人时跳过人脸管线
- 围栏：画面宽高各 80% 居中矩形

## 构建

- 必须 `CGO_ENABLED=1`
- Windows：MinGW/gcc + `download-deps.ps1` + `goversioninfo`
- macOS：clang + `download-deps-darwin.sh` + `build-darwin.sh`（须在 Mac 上执行）
- 嵌入拆分：`assets_embed_common.go` + 平台 build tag 文件
