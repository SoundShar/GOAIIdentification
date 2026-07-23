package main

import (
	"os"
	"time"
)

var quitChan = make(chan struct{}, 1)

func main() {
	if maybeRunNoticeUI() {
		return
	}

	console := os.Getenv("AIWEB_CONSOLE") == "1"
	if err := initLogger(console); err != nil {
		panic(err)
	}

	// 模型加载较慢：放到后台，先拉起 HTTPS 与托盘
	if os.Getenv("YKS_SKIP_DETECTOR") != "1" {
		go func() {
			if err := InitDetector(); err != nil {
				getLogger().Error("detector_init_failed", "error", err.Error())
			}
		}()
	}

	// 先完成 CA/提权与 Listen，再弹启动提示。
	// macOS 上若提示窗（osascript/JXA）与管理员授权同时出现，
	// 易触发 SecTrustSettings「no user interaction was possible」。
	if err := startHTTPServer(); err != nil {
		writeBootStatus(bootStatusFailed)
		spawnNoticeUI()
		// 给提示子进程时间弹出失败页（父进程随后退出，子进程继续驻留）
		time.Sleep(800 * time.Millisecond)
		os.Exit(1)
	}
	writeBootStatus(bootStatusReady)
	spawnNoticeUI()

	go func() {
		<-quitChan
		shutdownHTTPServer()
		os.Exit(0)
	}()

	getLogger().Info("application started")
	runTray()
}
