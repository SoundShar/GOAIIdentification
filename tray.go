package main

import (
	"embed"
	"runtime"

	"github.com/getlantern/systray"
)

//go:embed assets/icon.ico assets/icon.png
var trayIcons embed.FS

func trayIconData() []byte {
	iconName := "assets/icon.ico"
	if runtime.GOOS == "darwin" {
		iconName = "assets/icon.png"
	}

	data, err := trayIcons.ReadFile(iconName)
	if err != nil {
		getLogger().Error("tray icon load failed", "icon", iconName, "error", err.Error())
		return nil
	}
	return data
}

func onTrayReady() {
	systray.SetIcon(trayIconData())
	systray.SetTitle("AI Web")
	systray.SetTooltip("AI Web 本地服务运行中")

	quitItem := systray.AddMenuItem("退出", "关闭本地服务")
	go func() {
		<-quitItem.ClickedCh
		getLogger().Info("tray quit clicked")
		systray.Quit()
	}()
}

func onTrayExit() {
	select {
	case quitChan <- struct{}{}:
	default:
	}
}

func runTray() {
	systray.Run(onTrayReady, onTrayExit)
}
