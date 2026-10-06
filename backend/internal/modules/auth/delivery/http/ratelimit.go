package http

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginutil"
)

type rateLimiter struct {
	limit  int
	window time.Duration

	mu        sync.Mutex
	hits      map[string]*rateWindow
	lastSweep time.Time
}

type rateWindow struct {
	count   int
	resetAt time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		limit:     limit,
		window:    window,
		hits:      map[string]*rateWindow{},
		lastSweep: time.Now(),
	}
}

func (l *rateLimiter) allow(key string) bool {
	now := time.Now()
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
		l.hits[key] = &rateWindow{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	if w.count >= l.limit {
		return false
	}
	w.count++

	return true
}

func (l *rateLimiter) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if l.limit > 0 && !l.allow(c.ClientIP()) {
			ginutil.Error(c, apierr.RateLimited("too many attempts, try again later"))
			c.Abort()
			return
		}

		c.Next()
	}
}
