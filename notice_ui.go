package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func runNoticeUI() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return
	}
	port := ln.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)

	var status atomic.Value
	status.Store("starting")

	go func() {
		if waitForServiceReady(noticeHealthTimeout) {
			status.Store("success")
			return
		}
		status.Store("failed")
	}()

	closedCh := make(chan struct{})
	var closeOnce sync.Once
	signalClose := func() {
		closeOnce.Do(func() { close(closedCh) })
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(buildNoticeHTML()))
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(status.Load().(string)))
	})
	mux.HandleFunc("/close", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
		signalClose()
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(ln)
	}()

	hostCmd := startNoticeWindow(url)
	procDone := make(chan struct{})
	if hostCmd != nil {
		go func() {
			_ = hostCmd.Wait()
			close(procDone)
		}()
	} else {
		close(procDone)
	}

	select {
	case <-closedCh:
	case <-procDone:
	case <-time.After(noticeHealthTimeout + 5*time.Minute):
	}

	if hostCmd != nil && hostCmd.Process != nil {
		_ = hostCmd.Process.Kill()
	}
	_ = server.Close()
}

func buildNoticeHTML() string {
	iconData, err := trayIcons.ReadFile("assets/icon.png")
	iconSrc := ""
	if err == nil && len(iconData) > 0 {
		iconSrc = "data:image/png;base64," + base64.StdEncoding.EncodeToString(iconData)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"/>
<meta http-equiv="X-UA-Compatible" content="IE=edge"/>
<title>%s</title>
<style>
  html, body {
    margin: 0;
    padding: 0;
    width: 100%%;
    height: 100%%;
    overflow: hidden;
    background: #ffffff;
    color: #2b2b2b;
    font-family: "Microsoft YaHei UI", "PingFang SC", "Helvetica Neue", sans-serif;
    user-select: none;
  }
  .notice-shell {
    box-sizing: border-box;
    height: 100%%;
    padding: 16px 20px 14px;
    display: flex;
    flex-direction: column;
  }
  .notice-body {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 8px 4px;
  }
  .notice-icon {
    width: 52px;
    height: 52px;
    border-radius: 50%%;
    object-fit: contain;
    flex-shrink: 0;
  }
  .notice-icon-fallback {
    width: 52px;
    height: 52px;
    border-radius: 50%%;
    background: #1e5eff;
    flex-shrink: 0;
  }
  .notice-message {
    font-size: 15px;
    line-height: 1.4;
    color: #222;
  }
  .notice-loading {
    margin-top: 10px;
    width: 120px;
    height: 3px;
    border-radius: 2px;
    background: #e9eef8;
    overflow: hidden;
    position: relative;
  }
  .notice-loading > span {
    position: absolute;
    left: -40%%;
    top: 0;
    width: 40%%;
    height: 100%%;
    border-radius: 2px;
    background: #2f7bff;
    animation: notice-slide 1s ease-in-out infinite;
  }
  @keyframes notice-slide {
    0%% { left: -40%%; }
    100%% { left: 100%%; }
  }
  .notice-footer {
    display: flex;
    justify-content: flex-end;
    min-height: 36px;
    align-items: center;
  }
  .notice-ok {
    display: none;
    min-width: 88px;
    height: 34px;
    padding: 0 18px;
    border: 0;
    border-radius: 17px;
    background: #2f7bff;
    color: #fff;
    font-size: 14px;
    font-family: inherit;
    cursor: pointer;
    box-shadow: 0 0 0 3px rgba(47, 123, 255, 0.18);
  }
  .notice-ok:hover { background: #1f6cf0; }
  .notice-ok:focus { outline: none; }
  .is-done .notice-loading { display: none; }
  .is-done .notice-ok { display: inline-flex; align-items: center; justify-content: center; }
</style>
</head>
<body>
  <div class="notice-shell" id="notice-shell">
    <div class="notice-body">
      %s
      <div>
        <div class="notice-message" id="notice-message">%s</div>
        <div class="notice-loading" id="notice-loading"><span></span></div>
      </div>
    </div>
    <div class="notice-footer">
      <button class="notice-ok" id="notice-ok" type="button">%s</button>
    </div>
  </div>
  <script>
    (function () {
      var shell = document.getElementById('notice-shell');
      var message = document.getElementById('notice-message');
      var okBtn = document.getElementById('notice-ok');
      var done = false;

      function requestClose() {
        try {
          var xhr = new XMLHttpRequest();
          xhr.open('GET', '/close', false);
          xhr.send(null);
        } catch (e) {}
        try { window.close(); } catch (e2) {}
      }

      okBtn.onclick = requestClose;

      function applyStatus(status) {
        if (done) return;
        if (status === 'success') {
          done = true;
          message.textContent = %q;
          shell.className = 'notice-shell is-done';
          okBtn.focus();
          return;
        }
        if (status === 'failed') {
          done = true;
          message.textContent = %q;
          shell.className = 'notice-shell is-done';
          okBtn.focus();
        }
      }

      function poll() {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', '/status', true);
        xhr.onreadystatechange = function () {
          if (xhr.readyState === 4 && xhr.status === 200) {
            applyStatus((xhr.responseText || '').trim());
          }
        };
        xhr.send(null);
      }

      setInterval(poll, 200);
      poll();
    })();
  </script>
</body>
</html>`,
		noticeTitle,
		noticeIconHTML(iconSrc),
		noticeMsgStarting,
		noticeBtnOK,
		noticeMsgSuccess,
		noticeMsgFailed,
	)
}

func noticeIconHTML(iconSrc string) string {
	if iconSrc == "" {
		return `<div class="notice-icon-fallback"></div>`
	}
	return fmt.Sprintf(`<img class="notice-icon" alt="" src="%s"/>`, iconSrc)
}
