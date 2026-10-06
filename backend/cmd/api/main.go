package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/config"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/handler"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/repository/postgres"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/server"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "check the API liveness endpoint")
	flag.Parse()

	if *healthcheck {
		if err := checkHealth(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	level := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	repository := postgres.NewInvitationRepository(pool)
	invitationService := service.NewInvitationService(repository)
	authService := service.NewAuthService(postgres.NewAuthRepository(pool), cfg.SessionTTL)
	authHandler := handler.NewAuthHandler(authService, cfg.CookieSecure)
	managementService := service.NewInvitationManagementService(repository)
	guestService := service.NewGuestService(postgres.NewGuestRepository(pool))
	httpServer := server.New(
		fmt.Sprintf(":%d", cfg.Port),
		handler.NewHealthHandler(pool),
		handler.NewInvitationHandler(invitationService),
		authHandler,
		handler.NewInvitationManagementHandler(managementService, authHandler),
		handler.NewGuestHandler(guestService, authHandler),
		handler.NewInvitationMediaHandler(repository, authHandler),
	)

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("API listening", "address", httpServer.Addr)
		serverErrors <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-serverErrors:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func checkHealth() error {
	client := &http.Client{Timeout: 2 * time.Second}
	request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080/healthz", nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned %s", response.Status)
	}
	return nil
}
