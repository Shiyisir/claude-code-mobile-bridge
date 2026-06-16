package logging

import (
	"log/slog"
	"os"
)

var logger *slog.Logger

func Init() {
	level := slog.LevelInfo
	if os.Getenv("CLAUDE_PROXY_DEBUG") == "1" {
		level = slog.LevelDebug
	}
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}

func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}
