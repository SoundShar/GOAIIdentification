//go:build !windows

package main

import (
	"os/exec"
	"runtime"
)

func showServiceStartedNotice() {
	if runtime.GOOS != "darwin" {
		return
	}

	script := `display dialog "运行考试服务成功" with title "yks-tool" buttons {"确定"} default button 1 with icon note`
	_ = exec.Command("osascript", "-e", script).Start()
}
