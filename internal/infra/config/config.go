// Package config loads runtime configuration from environment variables.
package config

import "os"

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	DatabaseURL string
	LogLevel    string
	LogFormat   string
}

// NewConfig builds Config from environment variables.
func NewConfig() *Config {
	return &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		LogLevel:    getEnvOrDefault("LOG_LEVEL", "info"),
		LogFormat:   getEnvOrDefault("LOG_FORMAT", "text"),
	}
}

func getEnvOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
