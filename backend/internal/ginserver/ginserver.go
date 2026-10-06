package ginserver

import (
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/config"
)

func New(cfg config.Config) (*gin.Engine, error) {
	r := gin.New()

	// Behind nginx every connection comes from the proxy, so the client IP
	// (used by the auth rate limiter) must come from X-Real-IP. It is only
	// honoured when the connection itself is from a trusted proxy; X-Forwarded-For
	// is ignored because clients can prepend arbitrary values to it.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}
	r.RemoteIPHeaders = []string{"X-Real-IP"}

	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))
	return r, nil
}
