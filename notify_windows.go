//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	messageBoxProc = user32.NewProc("MessageBoxW")
)

const (
	mbOK              = 0x00000000
	mbIconInformation = 0x00000040
)

func showServiceStartedNotice() {
	title, err := syscall.UTF16PtrFromString("考试服务工具")
	if err != nil {
		return
	}
	text, err := syscall.UTF16PtrFromString("运行考试服务成功")
	if err != nil {
		return
	}
	messageBoxProc.Call(
		0,
		uintptr(unsafe.Pointer(text)),
		uintptr(unsafe.Pointer(title)),
		uintptr(mbOK|mbIconInformation),
	)
}
