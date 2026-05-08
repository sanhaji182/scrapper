package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DatabaseURL       string
	RedisAddr         string
	RedisPassword     string
	ProxyList         []string
	RequestTimeoutSec int
	WorkerConcurrency int
	AllowedOrigins    []string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RequestTimeoutSec: getEnvInt("REQUEST_TIMEOUT_SEC", 15),
		WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 5),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}

	rawProxy := getEnv("PROXY_LIST", "")
	if rawProxy != "" {
		cfg.ProxyList = strings.Split(rawProxy, ",")
	}

	rawOrigins := getEnv("ALLOWED_ORIGINS", "*")
	cfg.AllowedOrigins = strings.Split(rawOrigins, ",")

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if number, err := strconv.Atoi(value); err == nil {
			return number
		}
	}
	return fallback
}
