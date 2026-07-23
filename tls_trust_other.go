//go:build !windows && !darwin

package main

import (
	"crypto/x509"
	"fmt"
	"runtime"
)

// ensureSystemTrust 非 Windows/macOS 不支持自动写入系统信任库。
func ensureSystemTrust(_ *x509.Certificate, _ []byte, _ string) error {
	return fmt.Errorf("automatic root CA trust install is not supported on %s; use YKS_HTTP_ONLY=1 for local debug", runtime.GOOS)
}
