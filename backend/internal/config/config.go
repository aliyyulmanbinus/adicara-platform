package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            int
	DatabaseURL     string
	LogLevel        string
	CookieSecure    bool
	SessionTTL      time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	port, err := strconv.Atoi(envOrDefault("PORT", "8080"))
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be a valid TCP port")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	cookieDefault := "false"
	if envOrDefault("APP_ENV", "development") == "production" {
		cookieDefault = "true"
	}
	cookieSecure, err := strconv.ParseBool(envOrDefault("COOKIE_SECURE", cookieDefault))
	if err != nil {
		return Config{}, fmt.Errorf("COOKIE_SECURE must be true or false")
	}
	return Config{
		Port:            port,
		DatabaseURL:     databaseURL,
		LogLevel:        envOrDefault("LOG_LEVEL", "info"),
		CookieSecure:    cookieSecure,
		SessionTTL:      30 * 24 * time.Hour,
		ShutdownTimeout: 10 * time.Second,
	}, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
