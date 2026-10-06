package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/app"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/config"
)

const sessionCleanupInterval = time.Hour

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe /healthz on the local listener and exit (used by Docker)")
	flag.Parse()

	if *healthcheck {
		if err := probeHealth(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	a, err := app.Build(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer a.Pool.Close()

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go a.RunSessionCleanup(runCtx, sessionCleanupInterval)

	srv := &http.Server{
		Addr:              a.Config.HTTPAddr,
		Handler:           a.Handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("adicara-api listening on %s", a.Config.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-runCtx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

// probeHealth asks the already-running server on this host for /healthz, so
// the container is only "healthy" when the API answers and Postgres is up.
func probeHealth() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	addr := cfg.HTTPAddr
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + addr + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz returned %d", resp.StatusCode)
	}

	return nil
}
