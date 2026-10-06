// Package app is the composition root: it wires config, the database, and
// every domain service into one http.Handler. cmd/api/main.go (a persistent
// process) calls Build so the wiring lives in exactly one place.
package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/config"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/db"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/ginserver"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/migrate"
	authhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/delivery/http"
	authrepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/repository"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/auth/usecase"
	healthhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/health/delivery/http"
	profilehttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/profile/delivery/http"
	profileusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/profile/usecase"
)

type App struct {
	Config   config.Config
	Pool     *pgxpool.Pool
	Handler  http.Handler
	Identity *usecase.Service
}

func Build(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	// Built before touching the database so a bad config fails fast.
	ginEngine, err := ginserver.New(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	if cfg.MigrateOnStart {
		if err := migrate.Up(cfg.DatabaseURL); err != nil {
			pool.Close()
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}

	idn := usecase.New(authrepo.NewPostgresRepository(pool), cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL)

	healthhttp.RegisterRoutes(ginEngine, pool)
	authhttp.RegisterRoutes(ginEngine, cfg, idn)
	profileSvc := profileusecase.New(idn)
	profilehttp.RegisterRoutes(ginEngine, idn, profileSvc)

	mux := http.NewServeMux()
	mux.Handle("/v1/auth/", ginEngine)
	mux.Handle("/v1/me", ginEngine)
	mux.Handle("/healthz", ginEngine)
	var h http.Handler = mux

	return &App{
		Config:   cfg,
		Pool:     pool,
		Handler:  h,
		Identity: idn,
	}, nil
}

// RunSessionCleanup deletes expired refresh sessions once immediately and then
// every interval, until ctx is cancelled. Without it rotated sessions pile up
// in refresh_sessions forever.
func (a *App) RunSessionCleanup(ctx context.Context, interval time.Duration) {
	purge := func() {
		n, err := a.Identity.PurgeExpiredSessions(ctx)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("purge expired refresh sessions: %v", err)
			}
			return
		}
		if n > 0 {
			log.Printf("purged %d expired refresh sessions", n)
		}
	}

	purge()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			purge()
		}
	}
}
