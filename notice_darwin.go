//go:build darwin

package main

import (
	"fmt"
	"os/exec"
)

func showNativeNotice(title, msg string, success bool) {
	icon := "note"
	if !success {
		icon = "stop"
	}
	script := fmt.Sprintf(
		`display dialog %q with title %q buttons {"确定"} default button 1 with icon %s`,
		msg,
		title,
		icon,
	)
	cmd := exec.Command("osascript", "-e", script)
	_ = cmd.Run()
}
