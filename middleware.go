package main

import (
	"net/http"
	"net/url"
	"os"
	"strings"
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

var defaultCorsOrigins = []string{
	"https://yk.cetset.com",
	"https://test.cetset.com",
	"https://kspre.yks365.net",
	"https://ks.yks365.net",
	"https://local.cetset.com",
	"http://localhost:9627",
}

func allowedCorsOrigins() []string {
	raw := os.Getenv("YKS_CORS_ORIGIN")
	if raw == "" {
		return defaultCorsOrigins
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		return defaultCorsOrigins
	}
	return origins
}

func resolveCorsOrigin(requestOrigin string) (string, bool) {
	if requestOrigin == "" {
		return "", false
	}

	for _, allowedOrigin := range allowedCorsOrigins() {
		if requestOrigin == allowedOrigin {
			return requestOrigin, true
		}
	}

	parsedOrigin, err := url.Parse(requestOrigin)
	if err != nil || parsedOrigin.Hostname() == "" {
		return "", false
	}

	if parsedOrigin.Hostname() == "local.cetset.com" {
		return requestOrigin, true
	}

	return "", false
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestOrigin := r.Header.Get("Origin")
		if corsOrigin, ok := resolveCorsOrigin(requestOrigin); ok {
			w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

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
			"origin", r.Header.Get("Origin"),
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
