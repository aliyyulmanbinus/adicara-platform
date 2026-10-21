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
	plannerhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/delivery/http"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	plannerrepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/repository"
	plannerusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/usecase"
	financehttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/delivery/http"
	financerepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/repository"
	financeusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/usecase"
	timelinehttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/delivery/http"
	timelinerepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/repository"
	timelineusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/usecase"
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

	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
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

	// Planner: menu 01 (projects, sessions, dashboard) is the host; every other
	// planner module adds itself to it here when it is built:
	//   RegisterSummary           dashboard widgets
	//   RegisterTemplateAppliers  wizard templates
	//   RegisterSampleOverlays    the Wedding_Planner.pdf dataset of the sample project
	// Until a module is registered its widget is null and its template is not
	// offered. One registry instance serves every module (lead times, subjects).
	registry := domain.DefaultRegistry()
	plannerTx := plannerrepo.NewTxManager(pool)
	plannerSvc := plannerusecase.New(
		plannerrepo.NewPostgresRepository(pool), plannerTx, registry,
		plannerusecase.Options{MaxProjects: cfg.PlannerMaxProjects},
	)

	// Menu 02: timeline and administrative documents (two keys: timeline, admin_docs).
	timelineSvc := timelineusecase.New(timelinerepo.NewPostgresRepository(pool), plannerTx, registry, timelineusecase.Options{})

	// Menu 03: finance (scenarios, savings, budget lines; one key each: finance, budget, finance).
	// Its overlay must run after menu 02's (the Lamaran session date feeds the milestones) and
	// before the vendor module's, once that exists (it links vendors to the lines this overlay fills).
	financeSvc := financeusecase.New(financerepo.NewPostgresRepository(pool), plannerTx, registry, financeusecase.Options{})

	plannerSvc.RegisterSummary(timelineSvc.TimelineProvider(), timelineSvc.AdminDocsProvider(), financeSvc.Provider())
	plannerSvc.RegisterTemplateAppliers(timelineSvc.TimelineApplier(), timelineSvc.AdminDocsApplier(), financeSvc.Applier())
	plannerSvc.RegisterSampleOverlays(timelineSvc.TimelineOverlay(), timelineSvc.AdminDocsOverlay(), financeSvc.Overlay())

	plannerhttp.RegisterRoutes(ginEngine, idn, plannerSvc)
	timelinehttp.RegisterRoutes(ginEngine, idn, plannerSvc, timelineSvc)
	financehttp.RegisterRoutes(ginEngine, idn, plannerSvc, financeSvc)

	mux := http.NewServeMux()
	mux.Handle("/v1/auth/", ginEngine)
	mux.Handle("/v1/me", ginEngine)
	// app.go routes through an explicit ServeMux: without this line every
	// /v1/planner/* request answers 404.
	mux.Handle("/v1/planner/", ginEngine)
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
