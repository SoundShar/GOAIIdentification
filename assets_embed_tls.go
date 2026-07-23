package main

import "crypto/tls"

const publicServiceURL = "https://local.cetset.com:7986"

// loadTLSCertificate 确保本机 CA/叶子证就绪，并将根证安装到系统信任库。
// 用户拒绝提权或安装失败时返回 error（硬失败，不监听 HTTPS）。
func loadTLSCertificate() (tls.Certificate, error) {
	return ensureLocalTLS()
}
