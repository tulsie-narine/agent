package config

import (
	"os"
	"strconv"
	"time"
)

type APIConfig struct {
	DatabaseURL   string
	NATSUrl       string
	ServerPort    string
	TLSCertFile   string
	TLSKeyFile    string
	JWTSecret     string
	LogLevel      string
	RateLimitRPS  int
	MaxBatchSize  int
}

func Load() (*APIConfig, error) {
	cfg := &APIConfig{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://inventory:inventory123@localhost:5432/inventory?sslmode=disable"),
		NATSUrl:       getEnv("NATS_URL", "nats://localhost:4222"),
		ServerPort:    getEnv("API_PORT", "8080"),
		TLSCertFile:   getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:    getEnv("TLS_KEY_FILE", ""),
		JWTSecret:     getEnv("JWT_SECRET", "your-super-secure-jwt-secret-here-change-this-in-production"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		RateLimitRPS:  getEnvInt("RATE_LIMIT_RPS", 100),
		MaxBatchSize:  getEnvInt("MAX_BATCH_SIZE", 1000),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}