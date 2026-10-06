package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	HTTPAddr       string
	DatabaseURL    string
	JWTSecret      string
	AccessTTL      time.Duration
	RefreshTTL     time.Duration
	MigrateOnStart bool
	CORSOrigins    []string
	AuthRateLimit  int
	TrustedProxies []string
}

// defaultTrustedProxies covers loopback and private networks, which is where
// a reverse proxy (nginx in the docker network) reaches the API from. Only
// requests arriving from these addresses may set X-Real-IP.
const defaultTrustedProxies = "127.0.0.1,::1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,fc00::/7"

// defaultJWTSecret keeps `go run` working on a fresh checkout. Load refuses it
// (and anything else weak) whenever APP_ENV is not a development environment.
const defaultJWTSecret = "change-me-in-dev"

// minJWTSecretLen is the shortest secret accepted outside development: the
// HMAC key should be at least as long as the 32-byte SHA-256 output.
const minJWTSecretLen = 32

func Load() (Config, error) {
	_ = godotenv.Load()

	accessTTL, err := time.ParseDuration(getenv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return Config{}, fmt.Errorf("JWT_ACCESS_TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(getenv("JWT_REFRESH_TTL", "168h"))
	if err != nil {
		return Config{}, fmt.Errorf("JWT_REFRESH_TTL: %w", err)
	}

	dbHost := getenv("DB_HOST", "127.0.0.1")
	dbPort := getenv("DB_PORT", "5432")
	dbUser := getenv("DB_USERNAME", "postgres")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := getenv("DB_DATABASE", "adicara")
	dbSSL := getenv("DB_SSLMODE", "disable")

	var dbURL string
	if dbPass != "" {
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPass, dbHost, dbPort, dbName, dbSSL)
	} else {
		dbURL = fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s", dbUser, dbHost, dbPort, dbName, dbSSL)
	}

	// Override with DATABASE_URL if explicitly provided (e.g. from Platform as a Service)
	if envDBURL := os.Getenv("DATABASE_URL"); envDBURL != "" {
		dbURL = envDBURL
	}
	appEnv := getenv("APP_ENV", "development")
	secret := getenv("JWT_SECRET", defaultJWTSecret)
	if !isDevelopment(appEnv) {
		if err := checkJWTSecret(secret); err != nil {
			return Config{}, fmt.Errorf("JWT_SECRET (APP_ENV=%s): %w", appEnv, err)
		}
	}

	// 0 disables the limiter, which is useful for load tests.
	authRateLimit, err := strconv.Atoi(getenv("AUTH_RATE_LIMIT", "10"))
	if err != nil || authRateLimit < 0 {
		return Config{}, fmt.Errorf("AUTH_RATE_LIMIT must be a non-negative integer")
	}

	return Config{
		AppEnv:         appEnv,
		HTTPAddr:       httpAddr(),
		DatabaseURL:    dbURL,
		JWTSecret:      secret,
		AccessTTL:      accessTTL,
		RefreshTTL:     refreshTTL,
		MigrateOnStart: getenv("MIGRATE_ON_START", "true") == "true",
		CORSOrigins:    splitCSV(getenv("CORS_ORIGINS", "http://localhost:5173")),
		AuthRateLimit:  authRateLimit,
		TrustedProxies: splitCSV(getenv("TRUSTED_PROXIES", defaultTrustedProxies)),
	}, nil
}

// isDevelopment reports whether env may run with the weak default secret.
// Everything else, including a typo such as "prod", is treated as production
// so a misspelled APP_ENV fails closed instead of silently skipping the check.
func isDevelopment(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "", "development", "dev", "local", "test":
		return true
	}

	return false
}

// checkJWTSecret rejects a secret anyone could guess: unset, too short, or one
// of the placeholders shipped in the repository's example files. The secret
// itself is never put in the error.
func checkJWTSecret(secret string) error {
	lower := strings.ToLower(secret)
	switch {
	case secret == defaultJWTSecret || strings.Contains(lower, "change-me") || strings.HasPrefix(lower, "your-secret"):
		return errors.New("is unset or still a placeholder; set a random value, e.g. `openssl rand -hex 32`")
	case len(secret) < minJWTSecretLen:
		return fmt.Errorf("must be at least %d characters, got %d; e.g. `openssl rand -hex 32`", minJWTSecretLen, len(secret))
	}

	return nil
}

// httpAddr prefers PORT — the convention hosts like Railway/Heroku use
// to tell the app which port they expect it to listen on — falling back to
// HTTP_ADDR (or :8080) for local dev and anywhere PORT isn't set.
func httpAddr() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return getenv("HTTP_ADDR", ":8080")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
