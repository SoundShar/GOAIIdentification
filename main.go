package main

import (
	"os"
)

var quitChan = make(chan struct{}, 1)

func main() {
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

	if err := startHTTPServer(); err != nil {
		showStartupNotice(false)
		os.Exit(1)
	}
	showStartupNotice(true)

	go func() {
		<-quitChan
		shutdownHTTPServer()
		os.Exit(0)
	}()

	getLogger().Info("application started")
	runTray()
}
