package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func limitedEngine(t *testing.T, limit int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	if err := r.SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
		t.Fatal(err)
	}
	r.RemoteIPHeaders = []string{"X-Real-IP"}
	r.POST("/login", newRateLimiter(limit, time.Minute).middleware(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	return r
}

func post(r http.Handler, remoteAddr, realIP, forwardedFor string) int {
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = remoteAddr
	if realIP != "" {
		req.Header.Set("X-Real-IP", realIP)
	}
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w.Code
}

// Behind the proxy every connection has the proxy's address; the limiter must
// bucket by the client IP the proxy reports, not lock out everyone together.
func TestRateLimitBucketsByClientIPBehindTrustedProxy(t *testing.T) {
	r := limitedEngine(t, 2)
	const proxy = "10.0.0.5:4000"

	for i := 0; i < 2; i++ {
		if code := post(r, proxy, "203.0.113.1", ""); code != http.StatusNoContent {
			t.Fatalf("client A request %d: got %d", i+1, code)
		}
	}
	if code := post(r, proxy, "203.0.113.1", ""); code != http.StatusTooManyRequests {
		t.Fatalf("client A over limit: got %d, want 429", code)
	}
	if code := post(r, proxy, "203.0.113.2", ""); code != http.StatusNoContent {
		t.Fatalf("client B must have its own bucket: got %d", code)
	}
}

// A client that reaches the API directly must not be able to dodge the limit
// by rotating forged headers.
func TestRateLimitIgnoresForgedHeadersFromUntrustedPeer(t *testing.T) {
	r := limitedEngine(t, 2)
	const attacker = "198.51.100.7:5555"

	for i := 0; i < 2; i++ {
		if code := post(r, attacker, "1.1.1.1", ""); code != http.StatusNoContent {
			t.Fatalf("request %d: got %d", i+1, code)
		}
	}
	if code := post(r, attacker, "2.2.2.2", "3.3.3.3"); code != http.StatusTooManyRequests {
		t.Fatalf("rotated headers must not reset the bucket: got %d, want 429", code)
	}
}

// X-Forwarded-For is client-appendable, so a trusted proxy's value for it must
// not influence the bucket either.
func TestRateLimitIgnoresForwardedForFromTrustedProxy(t *testing.T) {
	r := limitedEngine(t, 1)
	const proxy = "10.0.0.5:4000"

	if code := post(r, proxy, "203.0.113.1", "9.9.9.1"); code != http.StatusNoContent {
		t.Fatalf("first request: got %d", code)
	}
	if code := post(r, proxy, "203.0.113.1", "9.9.9.2"); code != http.StatusTooManyRequests {
		t.Fatalf("changing X-Forwarded-For must not reset the bucket: got %d, want 429", code)
	}
}

func TestRateLimitZeroDisablesLimiter(t *testing.T) {
	r := limitedEngine(t, 0)
	for i := 0; i < 50; i++ {
		if code := post(r, "10.0.0.5:4000", "203.0.113.1", ""); code != http.StatusNoContent {
			t.Fatalf("request %d: got %d", i+1, code)
		}
	}
}
