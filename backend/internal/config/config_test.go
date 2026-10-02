package config

import "testing"

func TestCookieSecureConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		appEnv string
		value  string
		want   bool
	}{
		{"development default", "development", "", false},
		{"production default", "production", "", true},
		{"production HTTP override", "production", "false", false},
		{"production HTTPS override", "production", "true", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PORT", "8080")
			t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
			t.Setenv("APP_ENV", tt.appEnv)
			t.Setenv("COOKIE_SECURE", tt.value)
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.CookieSecure != tt.want {
				t.Fatalf("CookieSecure = %v, want %v", cfg.CookieSecure, tt.want)
			}
		})
	}
}

func TestRejectInvalidCookieSecure(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
	t.Setenv("COOKIE_SECURE", "invalid")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for invalid COOKIE_SECURE")
	}
}
