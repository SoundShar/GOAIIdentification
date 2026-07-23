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
- 围栏：画面宽高各 80% 居中矩形

## 构建

- 必须 `CGO_ENABLED=1`
- Windows：MinGW/gcc + `download-deps.ps1` + `goversioninfo`
- macOS：clang + `download-deps-darwin.sh` + `build-darwin.sh`（须在 Mac 上执行）；正式包再跑 `sign-notarize-darwin.sh`
- `MACOSX_DEPLOYMENT_TARGET=12.0`；Universal 用 `lipo`
- 嵌入拆分：`assets_embed_common.go` + 平台 build tag 文件
- App 模板：`packaging/macos/`（Info.plist、entitlements、AppIcon.icns）

## 启动提示 UI

- 文案：窗口标题「考试服务工具」（系统标题栏）；启动中「正在启动考试服务…」；成功「运行考试服务成功」；失败「启动考试服务失败」；按钮「确定」
- 页面内不再重复展示产品名标题，仅保留状态文案与确定按钮
- 实现：共用 HTML 页面；Windows 优先 Edge/Chrome `--app`（回退 mshta）；macOS 优先 JXA+WKWebView（回退 Chromium `--app`）
- 子进程参数：`--notice-ui`（仅弹窗，不启托盘/检测器）
- 时机：CA/提权与 Listen 完成后再拉起提示窗（避免与管理员授权对话框抢交互）；boot status 为 `ready`/`failed`；子进程轮询 `/api/health`

## 日志路径

- macOS：`~/Library/Logs/yks-tool/app.log`
- Windows：`%LOCALAPPDATA%\yks-tool\logs\app.log`

