//go:build !windows && !darwin

package main

func showNativeNotice(title, msg string, success bool) {
	_ = title
	_ = msg
	_ = success
}
