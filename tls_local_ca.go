package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	tlsLeafHost = "local.cetset.com"

	tlsCACertFile   = "ca.crt"
	tlsCAKeyFile    = "ca.key"
	tlsLeafCertFile = "local.cetset.com.crt"
	tlsLeafKeyFile  = "local.cetset.com.key"

	tlsCACommonName = "yks-tool Local CA"
	tlsCAOrg        = "yks-tool"

	tlsCARSABits       = 2048
	tlsLeafRSABits     = 2048
	tlsCAValidity      = 10 * 365 * 24 * time.Hour // ~10 年
	tlsLeafValidity    = 825 * 24 * time.Hour      // ~825 天
	tlsLeafRenewBefore = 30 * 24 * time.Hour       // 剩余 <30 天重签
)

// resolveSSLDir 返回本机 TLS 材料目录。
// Windows: %LOCALAPPDATA%\yks-tool\ssl\
// macOS: ~/Library/Application Support/yks-tool/ssl/
func resolveSSLDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "yks-tool", "ssl"), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "yks-tool", "ssl"), nil
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "yks-tool", "ssl"), nil
	}
}

// ensureLocalTLS 生成或复用本机 CA/叶子证，并将根证写入系统信任库（硬失败）。
func ensureLocalTLS() (tls.Certificate, error) {
	dir, err := resolveSSLDir()
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("resolve ssl dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return tls.Certificate{}, fmt.Errorf("create ssl dir: %w", err)
	}

	caCert, caKey, err := ensureCA(dir)
	if err != nil {
		return tls.Certificate{}, err
	}

	leafCertPEM, leafKeyPEM, err := ensureLeaf(dir, caCert, caKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	caCertPEM, err := os.ReadFile(filepath.Join(dir, tlsCACertFile))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("read ca cert: %w", err)
	}

	if err := ensureSystemTrust(caCert, caCertPEM, filepath.Join(dir, tlsCACertFile)); err != nil {
		getLogger().Error("tls_trust_install_failed",
			"error", err.Error(),
			"hint", "需授权安装证书才能启动",
		)
		return tls.Certificate{}, fmt.Errorf("需授权安装证书才能启动: %w", err)
	}

	pair, err := tls.X509KeyPair(leafCertPEM, leafKeyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("load local tls key pair: %w", err)
	}
	getLogger().Info("local_tls_ready",
		"dir", dir,
		"host", tlsLeafHost,
		"leaf_not_after", pairLeafNotAfter(pair),
	)
	return pair, nil
}

func pairLeafNotAfter(pair tls.Certificate) string {
	if len(pair.Certificate) == 0 {
		return ""
	}
	cert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return ""
	}
	return cert.NotAfter.Format(time.RFC3339)
}

func ensureCA(dir string) (*x509.Certificate, *rsa.PrivateKey, error) {
	certPath := filepath.Join(dir, tlsCACertFile)
	keyPath := filepath.Join(dir, tlsCAKeyFile)

	certPEM, certErr := os.ReadFile(certPath)
	keyPEM, keyErr := os.ReadFile(keyPath)
	if certErr == nil && keyErr == nil {
		cert, key, err := parseCertAndKey(certPEM, keyPEM)
		if err == nil && cert.IsCA && time.Now().Before(cert.NotAfter) {
			return cert, key, nil
		}
		getLogger().Info("local_ca_regenerate", "reason", "invalid_or_expired")
	}

	cert, key, err := generateCA()
	if err != nil {
		return nil, nil, err
	}
	if err := writeCertPEM(certPath, cert); err != nil {
		return nil, nil, err
	}
	if err := writeKeyPEM(keyPath, key); err != nil {
		return nil, nil, err
	}
	getLogger().Info("local_ca_generated", "path", certPath, "not_after", cert.NotAfter.Format(time.RFC3339))
	return cert, key, nil
}

func ensureLeaf(dir string, caCert *x509.Certificate, caKey *rsa.PrivateKey) ([]byte, []byte, error) {
	certPath := filepath.Join(dir, tlsLeafCertFile)
	keyPath := filepath.Join(dir, tlsLeafKeyFile)

	certPEM, certErr := os.ReadFile(certPath)
	keyPEM, keyErr := os.ReadFile(keyPath)
	if certErr == nil && keyErr == nil {
		cert, _, err := parseCertAndKey(certPEM, keyPEM)
		if err == nil && leafStillValid(cert, caCert) {
			return certPEM, keyPEM, nil
		}
		getLogger().Info("local_leaf_reissue", "reason", "missing_invalid_or_expiring")
	}

	cert, key, err := generateLeaf(caCert, caKey)
	if err != nil {
		return nil, nil, err
	}
	if err := writeCertPEM(certPath, cert); err != nil {
		return nil, nil, err
	}
	if err := writeKeyPEM(keyPath, key); err != nil {
		return nil, nil, err
	}
	certPEM, err = os.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read leaf cert: %w", err)
	}
	keyPEM, err = os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read leaf key: %w", err)
	}
	getLogger().Info("local_leaf_generated", "path", certPath, "not_after", cert.NotAfter.Format(time.RFC3339))
	return certPEM, keyPEM, nil
}

func leafStillValid(leaf, ca *x509.Certificate) bool {
	if leaf == nil || ca == nil {
		return false
	}
	now := time.Now()
	if now.Before(leaf.NotBefore) || !now.Before(leaf.NotAfter) {
		return false
	}
	if leaf.NotAfter.Sub(now) < tlsLeafRenewBefore {
		return false
	}
	if err := leaf.CheckSignatureFrom(ca); err != nil {
		return false
	}
	for _, name := range leaf.DNSNames {
		if name == tlsLeafHost {
			return true
		}
	}
	return leaf.Subject.CommonName == tlsLeafHost
}

func generateCA() (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, tlsCARSABits)
	if err != nil {
		return nil, nil, fmt.Errorf("generate ca key: %w", err)
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{tlsCAOrg},
			CommonName:   tlsCACommonName,
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(tlsCAValidity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("create ca cert: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, fmt.Errorf("parse ca cert: %w", err)
	}
	return cert, key, nil
}

func generateLeaf(caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, tlsLeafRSABits)
	if err != nil {
		return nil, nil, fmt.Errorf("generate leaf key: %w", err)
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{tlsCAOrg},
			CommonName:   tlsLeafHost,
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(tlsLeafValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{tlsLeafHost},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create leaf cert: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, fmt.Errorf("parse leaf cert: %w", err)
	}
	return cert, key, nil
}

func randomSerial() (*big.Int, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("serial: %w", err)
	}
	if serial.Sign() == 0 {
		serial = big.NewInt(1)
	}
	return serial, nil
}

func parseCertAndKey(certPEM, keyPEM []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil || certBlock.Type != "CERTIFICATE" {
		return nil, nil, fmt.Errorf("invalid cert pem")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("invalid key pem")
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		parsed, err2 := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err2 != nil {
			return nil, nil, fmt.Errorf("parse private key: %v / %w", err, err2)
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, fmt.Errorf("private key is not rsa")
		}
		key = rsaKey
	}
	return cert, key, nil
}

func writeCertPEM(path string, cert *x509.Certificate) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("write cert %s: %w", path, err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
		return fmt.Errorf("encode cert %s: %w", path, err)
	}
	return nil
}

func writeKeyPEM(path string, key *rsa.PrivateKey) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("write key %s: %w", path, err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		return fmt.Errorf("encode key %s: %w", path, err)
	}
	_ = f.Chmod(0o600)
	return nil
}
