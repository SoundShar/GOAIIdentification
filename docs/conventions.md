# 约定

## 命名

- 产物：
  - Windows：`yks-tool.exe`
  - macOS 开发：`yks-tool.app`（Universal；构建脚本会删除 lipo 中间二进制）
  - macOS 分发：`yks-tool-macos.zip`（内含 stapled `.app`）
- Bundle ID / CompanyName / LegalCopyright：`com.seaskylight.ykstool`
- 产品名 / 可执行文件：`yks-tool`；显示名：`考试服务工具`
- macOS：`LSUIElement=true`（菜单栏 Agent，不占 Dock）；`LSMinimumSystemVersion=12.0`
- 检测 JSON 键与 aiIdentification `meta.ts` 告警键一致

## 识别

- 单帧检测，无防抖（`changeTest` 帧累计逻辑不在本服务实现）
- YOLO 无人/多人时跳过人脸管线
- 围栏：四边各 0.2 边距（居中 60%×60% 内框）；**用五关键点**（左右眼、鼻、左右嘴）判定，任一点出界即 `rangeTestPC`（不用 YOLO 全身框，避免举手挡脸误报）
- 单人且人脸越界时只报 `rangeTestPC`，不跑低头/转头/换人；未检出脸则不做姿态与换人
- 低头：`pitch < -15`；转头：`|yaw| > 30`（两个独立判定，可同时触发；不做 roll）
- 换人：仅在单人、人脸在围栏内、无低头/转头、且 `face.score >= 0.7` 时比对；相似度 `< 0.4` 报换人；无人/多人/越界/低头/转头/低质量脸均不跑换人

## 构建

- 必须 `CGO_ENABLED=1`
- Windows：MinGW/gcc + `download-deps.ps1` + `goversioninfo`
- macOS：clang + `download-deps-darwin.sh` + `build-darwin.sh`（须在 Mac 上执行）；正式包再跑 `sign-notarize-darwin.sh`
- `MACOSX_DEPLOYMENT_TARGET=12.0`；Universal 用 `lipo`
- 嵌入拆分：`assets_embed_common.go` + 平台 build tag 文件
- App 模板：`packaging/macos/`（Info.plist、entitlements、AppIcon.icns）

## 启动提示 UI

- 文案：窗口标题「考试服务工具」；成功「运行考试服务成功」；失败「启动考试服务失败」；按钮「确定」
- 实现：Windows `MessageBoxW`；macOS `osascript display dialog`；其他平台无 GUI 时跳过
- 时机：CA/提权与 HTTPS Listen 完成后再弹窗（避免 macOS 上与管理员授权对话框抢交互）

## 日志路径

- macOS：`~/Library/Logs/yks-tool/app.log`
- Windows：`%LOCALAPPDATA%\yks-tool\logs\app.log`

