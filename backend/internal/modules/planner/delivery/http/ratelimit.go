package http

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginutil"
	authhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/delivery/http"
)

// userLimiter is the fixed-window limiter of the auth module (ratelimit.go),
// keyed by the authenticated user instead of the client IP. It is in-memory,
// so with several API replicas each one keeps its own counts.
type userLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu        sync.Mutex
	hits      map[string]*window
	lastSweep time.Time
}

type window struct {
	count   int
	resetAt time.Time
}

func newUserLimiter(limit int, per time.Duration) *userLimiter {
	return &userLimiter{limit: limit, window: per, now: time.Now, hits: map[string]*window{}, lastSweep: time.Now()}
}

func (l *userLimiter) allow(key string) bool {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastSweep) > l.window {
		for k, w := range l.hits {
			if now.After(w.resetAt) {
				delete(l.hits, k)
			}
		}
		l.lastSweep = now
	}

	w, ok := l.hits[key]
	if !ok || now.After(w.resetAt) {
		l.hits[key] = &window{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	if w.count >= l.limit {
		return false
	}
	w.count++

	return true
}

// middleware must run after BearerAuth. A limit <= 0 disables it.
func (l *userLimiter) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if l.limit > 0 && !l.allow(authhttp.CurrentUserID(c).String()) {
			ginutil.Error(c, apierr.RateLimited("too many requests, try again later"))
			c.Abort()
			return
		}

		c.Next()
	}
}
