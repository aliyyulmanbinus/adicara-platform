package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginutil"
)

// RegisterRoutes attaches /healthz onto a shared Gin engine.
func RegisterRoutes(r gin.IRouter, pool *pgxpool.Pool) {
	r.GET("/healthz", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			// Monitoring needs "dependency down", not a generic 500.
			ginutil.Error(c, apierr.Unavailable("database_unavailable", "cannot reach Postgres"))
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
