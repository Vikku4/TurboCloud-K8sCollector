package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port           string
	DBDSN          string
	RequestTimeout time.Duration
	LogLevel       string
}

func Load() Config {
	return Config{
		Port:           getEnv("PORT", "8080"),
		DBDSN:          mustEnv("DB_DSN"),
		RequestTimeout: getDurationEnv("REQUEST_TIMEOUT_SECONDS", 15) * time.Second,
		LogLevel:       getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var %s", key)
	}
	return v
}

func getDurationEnv(key string, defSeconds int) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return time.Duration(defSeconds)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("invalid %s=%s; using default %ds", key, v, defSeconds)
		return time.Duration(defSeconds)
	}
	return time.Duration(n)
}
