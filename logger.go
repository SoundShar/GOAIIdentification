package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
)

var appLogger *slog.Logger

// resolveLogDir 返回可写日志目录：
// macOS: ~/Library/Logs/yks-tool
// Windows: %LOCALAPPDATA%\yks-tool\logs
// 其它: ~/.local/share/yks-tool/logs
func resolveLogDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Logs", "yks-tool"), nil
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "yks-tool", "logs"), nil
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "yks-tool", "logs"), nil
	}
}

func initLogger(console bool) error {
	logDir, err := resolveLogDir()
	if err != nil {
		return err
	}
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
