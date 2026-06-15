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

	if os.Getenv("YKS_SKIP_DETECTOR") != "1" {
		if err := InitDetector(); err != nil {
			getLogger().Error("detector_init_failed", "error", err.Error())
		}
	}

	go func() {
		if err := startHTTPServer(); err != nil {
			os.Exit(1)
		}
	}()

	go func() {
		<-quitChan
		shutdownHTTPServer()
		os.Exit(0)
	}()

	getLogger().Info("application started")
	runTray()
}
