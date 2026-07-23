package main

import (
	"crypto/tls"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	noticeUIArg = "--notice-ui"

	noticeTitle          = "考试服务工具"
	noticeMsgStarting    = "正在启动考试服务…"
	noticeMsgSuccess     = "运行考试服务成功"
	noticeMsgFailed      = "启动考试服务失败"
	noticeBtnOK          = "确定"
	noticeHealthTimeout  = 60 * time.Second
	noticeHealthInterval = 200 * time.Millisecond

	bootStatusReady  = "ready"
	bootStatusFailed = "failed"
)

// maybeRunNoticeUI 若当前进程为启动提示子进程则运行 UI 并返回 true。
func maybeRunNoticeUI() bool {
	for _, arg := range os.Args[1:] {
		if arg == noticeUIArg {
			runNoticeUI()
			return true
		}
	}
	return false
}

// spawnNoticeUI 拉起启动提示子进程；调用方须先写好 boot status（ready/failed）。
func spawnNoticeUI() {
	exe, err := os.Executable()
	if err != nil {
		getLogger().Error("notice_ui_spawn_failed", "error", err.Error())
		return
	}
	cmd := exec.Command(exe, noticeUIArg)
	cmd.Env = os.Environ()
	configureNoticeUICmd(cmd)
	if err := cmd.Start(); err != nil {
		getLogger().Error("notice_ui_spawn_failed", "error", err.Error())
		return
	}
	getLogger().Info("notice_ui_spawned", "pid", cmd.Process.Pid)
	_ = cmd.Process.Release()
}

func noticeHealthURL() string {
	if os.Getenv("YKS_HTTP_ONLY") == "1" {
		return "http://127.0.0.1:7986/api/health"
	}
	return "https://127.0.0.1:7986/api/health"
}

func bootStatusPath() string {
	return filepath.Join(os.TempDir(), "yks-tool-boot-status")
}

func writeBootStatus(status string) {
	path := bootStatusPath()
	if err := os.WriteFile(path, []byte(status+"\n"), 0o600); err != nil {
		getLogger().Warn("boot_status_write_failed", "status", status, "error", err.Error())
	}
}

func readBootStatus() string {
	data, err := os.ReadFile(bootStatusPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func waitForServiceReady(timeout time.Duration) bool {
	client := &http.Client{
		Timeout: 800 * time.Millisecond,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 本机自签证书
		},
	}
	url := noticeHealthURL()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// 主进程 Listen 失败/拒绝授权时尽快结束，避免空等到超时
		if readBootStatus() == bootStatusFailed {
			return false
		}

		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(noticeHealthInterval)
	}
	return false
}
