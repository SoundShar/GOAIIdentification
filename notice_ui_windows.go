//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func configureNoticeUICmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x00000008, // DETACHED_PROCESS
	}
}

func startNoticeWindow(url string) *exec.Cmd {
	if cmd := startBrowserAppWindow(url); cmd != nil {
		return cmd
	}
	return startNoticeWindowMSHTA(url)
}

func startBrowserAppWindow(url string) *exec.Cmd {
	candidates := [][]string{
		edgeAppArgs(url),
		chromeAppArgs(url),
	}
	for _, args := range candidates {
		if len(args) == 0 {
			continue
		}
		cmd := exec.Command(args[0], args[1:]...)
		if err := cmd.Start(); err != nil {
			continue
		}
		return cmd
	}
	return nil
}

func edgeAppArgs(url string) []string {
	paths := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "Application", "msedge.exe"),
	}
	for _, path := range paths {
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return []string{path, "--app=" + url, "--window-size=420,190", "--disable-extensions"}
		}
	}
	return nil
}

func chromeAppArgs(url string) []string {
	paths := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "Application", "chrome.exe"),
	}
	for _, path := range paths {
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return []string{path, "--app=" + url, "--window-size=420,190", "--disable-extensions"}
		}
	}
	return nil
}

func startNoticeWindowMSHTA(url string) *exec.Cmd {
	hta := `<html>
<head>
<meta http-equiv="X-UA-Compatible" content="IE=edge"/>
<HTA:APPLICATION ID="yksNotice"
  APPLICATIONNAME="考试服务工具"
  BORDER="thin"
  BORDERSTYLE="static"
  CAPTION="yes"
  MAXIMIZEBUTTON="no"
  MINIMIZEBUTTON="no"
  SHOWINTASKBAR="yes"
  SINGLEINSTANCE="yes"
  SYSMENU="yes"
  SCROLL="no"
  WINDOWSTATE="normal"/>
<title>考试服务工具</title>
<script language="javascript">
window.resizeTo(430, 220);
window.moveTo((screen.width-430)/2, (screen.height-220)/2);
</script>
</head>
<body style="margin:0;padding:0;overflow:hidden;">
<iframe id="frame" src="` + url + `" width="100%" height="100%" frameborder="0"></iframe>
</body>
</html>`

	dir, err := os.MkdirTemp("", "yks-notice-*")
	if err != nil {
		return nil
	}
	htaPath := filepath.Join(dir, "notice.hta")
	if err := os.WriteFile(htaPath, []byte(hta), 0o644); err != nil {
		_ = os.RemoveAll(dir)
		return nil
	}

	cmd := exec.Command("mshta.exe", htaPath)
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(dir)
		return nil
	}
	go func() {
		_ = cmd.Wait()
		_ = os.RemoveAll(dir)
	}()
	return cmd
}
