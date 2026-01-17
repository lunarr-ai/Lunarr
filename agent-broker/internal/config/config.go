package config

import (
	"log/slog"
	"os"
	"strconv"
)

// Config holds application configuration from environment variables.
type Config struct {
	// Port is the HTTP server port.
	Port int
	// LogLevel is the minimum log level for logging.
	LogLevel slog.Level

	// Qdrant config
	QdrantHost   string
	QdrantPort   int
	QdrantAPIKey string
	QdrantUseTLS bool

	// Embedding config
	EmbeddingURL string
	EmbeddingDim int

	// Gemini config
	GeminiAPIKey string
	GeminiModel  string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:         getEnvInt("PORT", 8080),
		LogLevel:     getEnvLogLevel("LOG_LEVEL", slog.LevelInfo),
		QdrantHost:   getEnv("QDRANT_HOST", "localhost"),
		QdrantPort:   getEnvInt("QDRANT_PORT", 6334),
		QdrantAPIKey: getEnv("QDRANT_API_KEY", ""),
		QdrantUseTLS: getEnvBool("QDRANT_USE_TLS", false),
		EmbeddingURL: getEnv("EMBEDDING_URL", "http://localhost:8081"),
		EmbeddingDim: getEnvInt("EMBEDDING_DIM", 384),
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		GeminiModel:  getEnv("GEMINI_MODEL", "gemini-3-flash-preview"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	switch value {
	case "true", "1":
		return true
	case "false", "0":
		return false
	default:
		return defaultValue
	}
}

func getEnvLogLevel(key string, defaultValue slog.Level) slog.Level {
	value := getEnv(key, "")
	switch value {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO":
		return slog.LevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return defaultValue
	}
}
