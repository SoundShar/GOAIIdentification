//go:build !windows && !darwin

package main

import (
	"os/exec"
	"time"
)

func configureNoticeUICmd(cmd *exec.Cmd) {}

func startNoticeWindow(url string) *exec.Cmd {
	_ = url
	// 无 GUI 平台：短暂等待后通过 /close 结束（由 runNoticeUI 超时兜底）
	go func() {
		_ = waitForServiceReady(noticeHealthTimeout)
		time.Sleep(300 * time.Millisecond)
	}()
	return nil
}
