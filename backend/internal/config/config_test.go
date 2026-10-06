package config

import (
	"strings"
	"testing"
)

const strongSecret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func setEnv(t *testing.T, appEnv, secret string) {
	t.Helper()
	t.Setenv("APP_ENV", appEnv)
	t.Setenv("JWT_SECRET", secret)
	t.Setenv("DATABASE_URL", "postgres://u@127.0.0.1:5432/db")
}

func TestLoadAllowsDefaultSecretInDevelopment(t *testing.T) {
	for _, env := range []string{"", "development", "Dev", "local", "test"} {
		t.Run("APP_ENV="+env, func(t *testing.T) {
			setEnv(t, env, "")
			cfg, err := Load()
			if err != nil {
				t.Fatalf("development must accept the default secret: %v", err)
			}
			if cfg.JWTSecret != defaultJWTSecret {
				t.Fatalf("JWTSecret = %q, want the default", cfg.JWTSecret)
			}
		})
	}
}

func TestLoadRejectsWeakSecretOutsideDevelopment(t *testing.T) {
	tests := []struct {
		name, env, secret string
	}{
		{"unset", "production", ""},
		{"default", "production", defaultJWTSecret},
		{"placeholder from backend/.env.example", "production", "your-secret-key-at-least-32-chars-long"},
		{"placeholder from root .env.example", "production", "change-me-generate-a-real-secret-with-openssl-rand-hex-32"},
		{"too short", "production", "short-but-not-a-placeholder"},
		{"31 characters", "production", strings.Repeat("a", 31)},
		{"staging is not development", "staging", defaultJWTSecret},
		{"a typo fails closed", "prod", defaultJWTSecret},
		{"case-insensitive", "PRODUCTION", defaultJWTSecret},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.env, tt.secret)
			_, err := Load()
			if err == nil {
				t.Fatal("expected Load to reject the secret")
			}
			if !strings.Contains(err.Error(), "JWT_SECRET") {
				t.Fatalf("error should name JWT_SECRET: %v", err)
			}
			if tt.secret != "" && strings.Contains(err.Error(), tt.secret) {
				t.Fatalf("error must not echo the secret: %v", err)
			}
		})
	}
}

func TestLoadAcceptsStrongSecretInProduction(t *testing.T) {
	setEnv(t, "production", strongSecret)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppEnv != "production" || cfg.JWTSecret != strongSecret {
		t.Fatalf("unexpected config: env=%q", cfg.AppEnv)
	}

	// exactly the minimum length is fine
	setEnv(t, "production", strings.Repeat("k", minJWTSecretLen))
	if _, err := Load(); err != nil {
		t.Fatalf("a %d-character secret must be accepted: %v", minJWTSecretLen, err)
	}
}
