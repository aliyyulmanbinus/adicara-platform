package http

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/config"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
)

func RegisterRoutes(r gin.IRouter, cfg config.Config, svc *usecase.Service) {
	h := NewHandler(svc)
	limiter := newRateLimiter(cfg.AuthRateLimit, time.Minute)

	auth := r.Group("/v1/auth")
	{
		credentials := auth.Group("")
		credentials.Use(limiter.middleware())
		credentials.POST("/register", h.Register)
		credentials.POST("/login", h.Login)
		credentials.POST("/refresh", h.Refresh)
		credentials.POST("/logout", h.Logout)

		protected := auth.Group("")
		protected.Use(BearerAuth(svc))
		protected.POST("/password", h.ChangePassword)
	}
}
