package main

import (
	_ "embed"
	"crypto/tls"
	"fmt"
)

//go:embed embeddata/ssl/local.sharas.cn_bundle.crt
var embedTLSCert []byte

//go:embed embeddata/ssl/local.sharas.cn.key
var embedTLSKey []byte

const publicServiceURL = "https://local.sharas.cn:7986"

func loadTLSCertificate() (tls.Certificate, error) {
	if len(embedTLSCert) == 0 || len(embedTLSKey) == 0 {
		return tls.Certificate{}, fmt.Errorf("embedded TLS cert/key is empty, run build script to copy ssl/ into embeddata/ssl/")
	}
	cert, err := tls.X509KeyPair(embedTLSCert, embedTLSKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("load TLS key pair: %w", err)
	}
	return cert, nil
}
