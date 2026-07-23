//go:build darwin

package main

import (
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const systemKeychain = "/Library/Keychains/System.keychain"

// ensureSystemTrust 检测 macOS 系统钥匙串信任；未信任则 osascript 提权安装。
// 用户取消管理员密码或安装失败 → 返回 error（硬失败）。
func ensureSystemTrust(caCert *x509.Certificate, _ []byte, caCertPath string) error {
	thumb := strings.ToUpper(hex.EncodeToString(sha1Fingerprint(caCert)))
	if isCATrustedInSystem(thumb) {
		getLogger().Info("tls_trust_already_installed", "thumbprint", thumb)
		return nil
	}

	getLogger().Info("tls_trust_install_start", "keychain", systemKeychain, "path", caCertPath, "thumbprint", thumb)
	if err := elevateAddTrustedCert(caCertPath); err != nil {
		return fmt.Errorf("install trusted ca (admin denied or failed): %w", err)
	}
	// 钥匙串写入后偶发短暂不可见，短重试再判定失败
	if !waitCATrustedInSystem(thumb, 5*time.Second) {
		return fmt.Errorf("ca not found in system keychain after install (thumbprint %s)", thumb)
	}
	getLogger().Info("tls_trust_install_ok", "thumbprint", thumb)
	return nil
}

func waitCATrustedInSystem(thumbprint string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if isCATrustedInSystem(thumbprint) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func sha1Fingerprint(cert *x509.Certificate) []byte {
	sum := sha1.Sum(cert.Raw)
	return sum[:]
}

func isCATrustedInSystem(thumbprint string) bool {
	cmd := exec.Command("security", "find-certificate", "-a", "-Z", systemKeychain)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	// security -Z 输出形如：SHA-1 hash: AABBCC...
	upper := strings.ToUpper(string(out))
	normalized := strings.ReplaceAll(thumbprint, ":", "")
	return strings.Contains(upper, thumbprint) || strings.Contains(upper, normalized)
}

// elevateAddTrustedCert 通过 AppleScript 请求管理员权限写入系统钥匙串并标记 trustRoot。
func elevateAddTrustedCert(caCertPath string) error {
	if strings.Contains(caCertPath, `"`) || strings.Contains(systemKeychain, `"`) {
		return fmt.Errorf("ca cert path contains quote")
	}
	script := fmt.Sprintf(
		`do shell script "security add-trusted-cert -d -r trustRoot -k " & quoted form of "%s" & " " & quoted form of "%s" with administrator privileges`,
		systemKeychain,
		caCertPath,
	)
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("osascript add-trusted-cert failed: %s", msg)
	}
	return nil
}
