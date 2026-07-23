//go:build !windows && !darwin

package main

func showServiceStartedNotice() {
	// Linux / other: no native notice UI
}
