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
	guesthttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/delivery/http"
	guestrepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/repository"
	guestusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/usecase"
	rundownhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/delivery/http"
	rundownrepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/repository"
	rundownusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/usecase"
	timelinehttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/delivery/http"
	timelinerepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/repository"
	timelineusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/usecase"
	vendorhttp "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/delivery/http"
	vendorrepo "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/repository"
	vendorusecase "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/usecase"
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
	// Its overlay only fills lines the budget applier already made, so it needs no other overlay first.
	financeSvc := financeusecase.New(financerepo.NewPostgresRepository(pool), plannerTx, registry, financeusecase.Options{})

	// Menu 04: vendors and the shopping list (provider vendors, applier shopping, overlays vendors and shopping).
	// Its overlays point sample budget lines and one sample task at the vendors, and the shopping overlay needs the items
	// its applier made. Those rows come from the appliers, which the host runs before any overlay, so the overlays need no
	// order among themselves (swapping the finance and vendor overlays leaves every sample table unchanged; checked
	// 2026-10-31). The shopping applier does run after the budget one, by TemplateKeys order.
	vendorSvc := vendorusecase.New(vendorrepo.NewPostgresRepository(pool), plannerTx, registry, vendorusecase.Options{})

	// Menu 05: guests, RSVP and the reception desk (provider guests, overlay guests; no template: the wizard creates no
	// guests). Its overlay depends on no other. Options.Invitations stays empty until the invitation module
	// exists, so no guest has a personal link yet (05 BE 5); that module will also take guestSvc.RSVPWriter().
	guestSvc := guestusecase.New(guestrepo.NewPostgresRepository(pool), plannerTx, registry, guestusecase.Options{})

	// Menu 06: the rundown and the playlist (provider rundown, appliers rundown and playlist; no overlay: the sample
	// project holds the template, 33 agendas and 30 songs, all pending). The appliers run after every other one
	// (TemplateKeys order) and need no row of another module.
	rundownSvc := rundownusecase.New(rundownrepo.NewPostgresRepository(pool), plannerTx, registry, rundownusecase.Options{})

	plannerSvc.RegisterSummary(
		timelineSvc.TimelineProvider(), timelineSvc.AdminDocsProvider(), financeSvc.Provider(), vendorSvc.Provider(), guestSvc.Provider(),
		rundownSvc.Provider(),
	)
	plannerSvc.RegisterTemplateAppliers(
		timelineSvc.TimelineApplier(), timelineSvc.AdminDocsApplier(), financeSvc.Applier(), vendorSvc.Applier(),
		rundownSvc.RundownApplier(), rundownSvc.PlaylistApplier(),
	)
	plannerSvc.RegisterSampleOverlays(
		timelineSvc.TimelineOverlay(), timelineSvc.AdminDocsOverlay(), financeSvc.Overlay(),
		vendorSvc.VendorsOverlay(), vendorSvc.ShoppingOverlay(), guestSvc.GuestsOverlay(),
	)

	plannerhttp.RegisterRoutes(ginEngine, idn, plannerSvc)
	timelinehttp.RegisterRoutes(ginEngine, idn, plannerSvc, timelineSvc)
	financehttp.RegisterRoutes(ginEngine, idn, plannerSvc, financeSvc)
	vendorhttp.RegisterRoutes(ginEngine, idn, plannerSvc, vendorSvc)
	guesthttp.RegisterRoutes(ginEngine, idn, plannerSvc, guestSvc)
	rundownhttp.RegisterRoutes(ginEngine, idn, plannerSvc, rundownSvc)

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
