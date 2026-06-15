package main

import (
	"context"
	"errors"
	"net/http"
	"time"
)

const serverAddr = "127.0.0.1:7986"

var httpServer *http.Server

func newMux() http.Handler {
	mux := http.NewServeMux()
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

	getLogger().Info("http server starting", "addr", serverAddr)
	err := httpServer.ListenAndServe()
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
