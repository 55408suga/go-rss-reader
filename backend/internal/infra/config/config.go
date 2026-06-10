// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strings"
)

// defaultCORSOrigin is the Next.js dev server, used when CORS_ALLOWED_ORIGINS
// is unset so local development works out of the box.
const defaultCORSOrigin = "http://localhost:3000"

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	DatabaseURL        string
	LogLevel           string
	LogFormat          string
	CORSAllowedOrigins []string
}

// NewConfig builds Config from environment variables.
func NewConfig() *Config {
	return &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		LogLevel:           getEnvOrDefault("LOG_LEVEL", "info"),
		LogFormat:          getEnvOrDefault("LOG_FORMAT", "text"),
		CORSAllowedOrigins: parseCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS")),
	}
}

// parseCORSOrigins splits a comma-separated allowlist, trimming each entry and
// dropping blanks. An empty or all-blank value falls back to the dev default.
func parseCORSOrigins(raw string) []string {
	origins := make([]string, 0)
	for part := range strings.SplitSeq(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	if len(origins) == 0 {
		return []string{defaultCORSOrigin}
	}
	return origins
}

func getEnvOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
