# 约定

## 命名

- 产物：
  - Windows：`yks-tool.exe`
  - macOS 开发：`yks-tool.app`（Universal），中间二进制 `yks-tool-darwin-arm64` / `yks-tool-darwin-amd64` / `yks-tool-darwin-universal`
  - macOS 分发：`yks-tool-macos.zip`（内含 stapled `.app`）
- Bundle ID / CompanyName / LegalCopyright：`com.seaskylight.ykstool`
- 产品名 / 可执行文件：`yks-tool`；显示名：`考试服务工具`
- macOS：`LSUIElement=true`（菜单栏 Agent，不占 Dock）；`LSMinimumSystemVersion=12.0`
- 检测 JSON 键与 aiIdentification `meta.ts` 告警键一致

## 识别

- 单帧检测，无防抖（`changeTest` 帧累计逻辑不在本服务实现）
- YOLO 无人/多人时跳过人脸管线
- 围栏：画面宽高各 80% 居中矩形

## 构建

- 必须 `CGO_ENABLED=1`
- Windows：MinGW/gcc + `download-deps.ps1` + `goversioninfo`
- macOS：clang + `download-deps-darwin.sh` + `build-darwin.sh`（须在 Mac 上执行）；正式包再跑 `sign-notarize-darwin.sh`
- `MACOSX_DEPLOYMENT_TARGET=12.0`；Universal 用 `lipo`
- 嵌入拆分：`assets_embed_common.go` + 平台 build tag 文件
- App 模板：`packaging/macos/`（Info.plist、entitlements、AppIcon.icns）

## 日志路径

- macOS：`~/Library/Logs/yks-tool/app.log`
- Windows：`%LOCALAPPDATA%\yks-tool\logs\app.log`
