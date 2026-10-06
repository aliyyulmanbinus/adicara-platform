package ginserver

import (
	"testing"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/config"
)

func testConfig(proxies ...string) config.Config {
	return config.Config{CORSOrigins: []string{"http://localhost:4321"}, TrustedProxies: proxies}
}

func TestNewRejectsInvalidTrustedProxies(t *testing.T) {
	if _, err := New(testConfig("not-a-cidr")); err == nil {
		t.Fatal("invalid TRUSTED_PROXIES must be rejected at startup")
	}
}

func TestNewAcceptsValidTrustedProxies(t *testing.T) {
	if _, err := New(testConfig("127.0.0.1", "10.0.0.0/8", "fc00::/7")); err != nil {
		t.Fatalf("valid proxies rejected: %v", err)
	}
}
