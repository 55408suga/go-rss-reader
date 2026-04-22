package logger

import (
	"log/slog"
	"os"
	"strings"

	"rss_reader/internal/infra/config"
)

// New builds a configured slog logger from application config.
func New(cfg *config.Config) *slog.Logger {
	if cfg == nil {
		cfg = &config.Config{}
	}

	level := parseLevel(cfg.LogLevel)
	opts := &slog.HandlerOptions{Level: level}

	if strings.EqualFold(cfg.LogFormat, "json") {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}

	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
