package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const logDir = "logs"

var appLogger *slog.Logger

func initLogger(console bool) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logPath := filepath.Join(logDir, "app.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	writers := []io.Writer{logFile}
	if console {
		writers = append(writers, os.Stdout)
	}

	handler := slog.NewTextHandler(io.MultiWriter(writers...), &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	appLogger = slog.New(handler)
	slog.SetDefault(appLogger)

	appLogger.Info("logger initialized", "log_file", logPath)
	return nil
}

func getLogger() *slog.Logger {
	if appLogger == nil {
		return slog.Default()
	}
	return appLogger
}
