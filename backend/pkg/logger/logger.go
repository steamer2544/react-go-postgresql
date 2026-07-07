// Package logger provides a structured logger backed by slog.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a configured *slog.Logger. The level string must be one of
// "debug", "info", "warn", "error" (case-insensitive).
func New(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	}))
}
