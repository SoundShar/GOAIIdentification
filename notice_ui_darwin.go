//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func configureNoticeUICmd(cmd *exec.Cmd) {}

func startNoticeWindow(url string) *exec.Cmd {
	// 优先 JXA+WKWebView，不依赖本机是否安装 Chrome，且与 Windows HTML 页一致
	if cmd := startNoticeWindowJXA(url); cmd != nil {
		return cmd
	}
	return startChromiumAppWindow(url)
}

func startChromiumAppWindow(url string) *exec.Cmd {
	candidates := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	}
	for _, bin := range candidates {
		if st, err := os.Stat(bin); err == nil && !st.IsDir() {
			cmd := exec.Command(bin, "--app="+url, "--window-size=420,190", "--disable-extensions")
			if err := cmd.Start(); err != nil {
				continue
			}
			return cmd
		}
	}
	return nil
}

func startNoticeWindowJXA(url string) *exec.Cmd {
	script := buildNoticeJXA(url)
	cmd := exec.Command("osascript", "-l", "JavaScript", "-e", script)
	if err := cmd.Start(); err != nil {
		return nil
	}
	return cmd
}

func buildNoticeJXA(url string) string {
	safeURL := strings.ReplaceAll(url, `\`, `\\`)
	safeURL = strings.ReplaceAll(safeURL, `"`, `\"`)

	return fmt.Sprintf(`
ObjC.import('Cocoa');
ObjC.import('WebKit');

var pageURL = "%s";
var app = $.NSApplication.sharedApplication;
app.setActivationPolicy($.NSApplicationActivationPolicyAccessory);

var style = $.NSWindowStyleMaskTitled | $.NSWindowStyleMaskClosable;
var rect = $.NSMakeRect(0, 0, 420, 180);
var win = $.NSWindow.alloc.initWithContentRectStyleMaskBackingDefer(
  rect,
  style,
  $.NSBackingStoreBuffered,
  false
);
win.setTitle("考试服务工具");
win.setReleasedWhenClosed(false);
win.center;

var config = $.WKWebViewConfiguration.alloc.init;
var webView = $.WKWebView.alloc.initWithFrameConfiguration($.NSMakeRect(0, 0, 420, 180), config);
webView.setAutoresizingMask($.NSViewWidthSizable | $.NSViewHeightSizable);
win.contentView = webView;

var request = $.NSURLRequest.requestWithURL($.NSURL.URLWithString(pageURL));
webView.loadRequest(request);
win.makeKeyAndOrderFront(app);
app.activateIgnoringOtherApps(true);

ObjC.registerSubclass({
  name: 'YksNoticeAppDelegate',
  protocols: ['NSApplicationDelegate'],
  methods: {
    'applicationShouldTerminateAfterLastWindowClosed:': function (sender) {
      return true;
    }
  }
});

app.setDelegate($.YksNoticeAppDelegate.alloc.init);
app.run();
`, safeURL)
}
