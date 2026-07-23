//go:build windows

package main

import (
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// ensureSystemTrust 检测 Windows Root 库；未安装则 UAC 提权 certutil 写入。
// 用户取消 UAC 或安装失败 → 返回 error（硬失败）。
func ensureSystemTrust(caCert *x509.Certificate, _ []byte, caCertPath string) error {
	thumb := strings.ToUpper(hex.EncodeToString(sha1Fingerprint(caCert)))
	if isRootCAInstalled(thumb) {
		getLogger().Info("tls_trust_already_installed", "thumbprint", thumb)
		return nil
	}

	getLogger().Info("tls_trust_install_start", "store", "Root", "path", caCertPath, "thumbprint", thumb)
	if err := elevateCertutilAddRoot(caCertPath); err != nil {
		return fmt.Errorf("install root ca (uac denied or failed): %w", err)
	}
	// 证书库写入后偶发短暂不可见，短重试再判定失败
	if !waitRootCAInstalled(thumb, 5*time.Second) {
		return fmt.Errorf("root ca not found in store after install (thumbprint %s)", thumb)
	}
	getLogger().Info("tls_trust_install_ok", "thumbprint", thumb)
	return nil
}

func waitRootCAInstalled(thumbprint string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if isRootCAInstalled(thumbprint) {
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

func isRootCAInstalled(thumbprint string) bool {
	cmd := exec.Command("certutil", "-store", "Root", thumbprint)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	upper := strings.ToUpper(string(out))
	return strings.Contains(upper, thumbprint)
}

// elevateCertutilAddRoot 通过 PowerShell Start-Process -Verb RunAs 提权执行 certutil。
func elevateCertutilAddRoot(caCertPath string) error {
	// 路径单引号转义，避免 PowerShell 注入
	escaped := strings.ReplaceAll(caCertPath, "'", "''")
	ps := fmt.Sprintf(
		"$p = Start-Process -FilePath 'certutil' -ArgumentList @('-addstore','-f','Root','%s') -Verb RunAs -Wait -PassThru; if ($null -eq $p) { exit 1 }; exit $p.ExitCode",
		escaped,
	)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("certutil elevate failed: %s", msg)
	}
	return nil
}
