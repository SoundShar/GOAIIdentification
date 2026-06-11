package main

import (
	"net/http"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	bytesOut    int
}

func (rec *responseRecorder) WriteHeader(statusCode int) {
	if rec.wroteHeader {
		return
	}
	rec.status = statusCode
	rec.wroteHeader = true
	rec.ResponseWriter.WriteHeader(statusCode)
}

func (rec *responseRecorder) Write(data []byte) (int, error) {
	if !rec.wroteHeader {
		rec.WriteHeader(http.StatusOK)
	}
	n, err := rec.ResponseWriter.Write(data)
	rec.bytesOut += n
	return n, err
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(rec, r)

		bytesIn := r.ContentLength
		if bytesIn < 0 {
			bytesIn = 0
		}

		getLogger().Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes_in", bytesIn,
			"bytes_out", rec.bytesOut,
			"remote", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}

func chainMiddleware(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
