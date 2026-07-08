package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"os"
	"time"
)

const serverAddr = "127.0.0.1:7986"

var httpServer *http.Server

func newMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/init", handleInit)
	mux.HandleFunc("/api/upload", handleUpload)

	return chainMiddleware(mux, loggingMiddleware, corsMiddleware)
}

func startHTTPServer() error {
	httpServer = &http.Server{
		Addr:              serverAddr,
		Handler:           newMux(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if os.Getenv("YKS_HTTP_ONLY") == "1" {
		getLogger().Info("http server starting", "addr", serverAddr, "tls", false)
		return serveHTTPListener(false)
	}

	cert, err := loadTLSCertificate()
	if err != nil {
		return err
	}

	httpServer.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	getLogger().Info("https server starting",
		"addr", serverAddr,
		"tls", true,
		"public_url", publicServiceURL,
	)
	return serveHTTPListener(true)
}

func serveHTTPListener(useTLS bool) error {
	var (
		ln  net.Listener
		err error
	)

	if useTLS {
		ln, err = tls.Listen("tcp", serverAddr, httpServer.TLSConfig)
	} else {
		ln, err = net.Listen("tcp", serverAddr)
	}
	if err != nil {
		getLogger().Error("http server listen failed", "error", err.Error())
		return err
	}

	showServiceStartedNotice()

	err = httpServer.Serve(ln)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		getLogger().Error("http server failed", "error", err.Error())
		return err
	}
	return nil
}

func shutdownHTTPServer() {
	if httpServer == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	getLogger().Info("http server shutting down")
	if err := httpServer.Shutdown(ctx); err != nil {
		getLogger().Error("http server shutdown failed", "error", err.Error())
	}
}
