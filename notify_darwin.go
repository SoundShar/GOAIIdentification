//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func showServiceStartedNotice() {
	script := buildMacNoticeScript(macNoticeIconPath())
	_ = exec.Command("osascript", "-e", script).Start()
}

func buildMacNoticeScript(iconPath string) string {
	const (
		title   = "考试服务工具"
		message = "运行考试服务成功"
	)
	if iconPath == "" {
		return fmt.Sprintf(
			`display dialog %s with title %s buttons {"确定"} default button 1`,
			appleScriptString(message),
			appleScriptString(title),
		)
	}
	return fmt.Sprintf(
		`display dialog %s with title %s buttons {"确定"} default button 1 with icon (POSIX file %s)`,
		appleScriptString(message),
		appleScriptString(title),
		appleScriptString(iconPath),
	)
}

func appleScriptString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// macNoticeIconPath 优先使用 .app 内 AppIcon.icns，其次 .app 本身，最后解压内嵌 PNG。
func macNoticeIconPath() string {
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		if idx := strings.Index(exe, ".app/Contents/MacOS/"); idx >= 0 {
			appRoot := exe[:idx+4] // includes ".app"
			icns := filepath.Join(appRoot, "Contents", "Resources", "AppIcon.icns")
			if st, err := os.Stat(icns); err == nil && !st.IsDir() {
				return icns
			}
			return appRoot
		}
	}

	if path := writeEmbeddedNoticeIcon(); path != "" {
		return path
	}

	// 开发目录回退：仓库内 packaging 图标
	candidates := []string{
		filepath.Join("packaging", "macos", "AppIcon.icns"),
		filepath.Join("assets", "icon.png"),
	}
	for _, candidate := range candidates {
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			abs, err := filepath.Abs(candidate)
			if err == nil {
				return abs
			}
			return candidate
		}
	}
	return ""
}

func writeEmbeddedNoticeIcon() string {
	data, err := trayIcons.ReadFile("assets/icon.png")
	if err != nil || len(data) == 0 {
		return ""
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(cacheDir, "yks-tool")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	path := filepath.Join(dir, "notice-icon.png")
	if st, err := os.Stat(path); err == nil && st.Size() == int64(len(data)) {
		return path
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return ""
	}
	return path
}
