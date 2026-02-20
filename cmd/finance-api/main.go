package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ahmethakanbesel/finance-api/internal/config"
	"github.com/ahmethakanbesel/finance-api/internal/job"
	"github.com/ahmethakanbesel/finance-api/internal/platform/sqlite"
	"github.com/ahmethakanbesel/finance-api/internal/price"
	"github.com/ahmethakanbesel/finance-api/internal/rate"
	jobrepo "github.com/ahmethakanbesel/finance-api/internal/repository/job"
	pricerepo "github.com/ahmethakanbesel/finance-api/internal/repository/price"
	raterepo "github.com/ahmethakanbesel/finance-api/internal/repository/rate"
	"github.com/ahmethakanbesel/finance-api/internal/scraper"
	"github.com/ahmethakanbesel/finance-api/internal/scraper/isyatirim"
	"github.com/ahmethakanbesel/finance-api/internal/scraper/tefas"
	"github.com/ahmethakanbesel/finance-api/internal/scraper/yahoo"
	"github.com/ahmethakanbesel/finance-api/internal/server"
)

func main() {
	cfg := config.Load()

	// Root context: cancelled on SIGINT/SIGTERM so in-flight scraper workers
	// stop promptly during graceful shutdown.
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// Open database
	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	// Repositories
	priceRepo := pricerepo.NewRepository(db.DB)
	jobRepo := jobrepo.NewRepository(db.DB)
	rateRepo := raterepo.NewRepository(db.DB)

	// Scraper registry
	registry := scraper.NewRegistry()
	registry.Register(tefas.New(tefas.WithWorkers(cfg.Workers)))
	registry.Register(yahoo.New(yahoo.WithWorkers(cfg.Workers)))
	registry.Register(isyatirim.New(isyatirim.WithWorkers(cfg.Workers)))

	// Services
	rateSvc := rate.NewService(rateRepo)
	jobSvc := job.NewService(jobRepo)
	priceSvc := price.NewService(priceRepo, jobRepo, registry, rateSvc)

	// Worker pool: picks up pending jobs in the background
	pool := job.NewWorkerPool(jobRepo, priceSvc, cfg.Workers)
	priceSvc.SetNotify(pool.Notify)
	poolDone := make(chan struct{})
	go func() {
		pool.Run(rootCtx)
		close(poolDone)
	}()

	// Re-queue interrupted jobs (pending/running) so workers pick them up.
	if err := jobSvc.RecoverStaleJobs(rootCtx); err != nil {
		slog.Error("failed to recover stale jobs", "error", err)
	}
	pool.Notify()

	// HTTP server — rootCtx is used as BaseContext so every request context
	// inherits from it and is cancelled on shutdown.
	srv := server.New(rootCtx, cfg.Port, priceSvc, jobSvc)

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("server started", "port", cfg.Port)
	<-done

	// Cancel root context first so in-flight requests (and their scraper
	// workers) begin winding down immediately.
	rootCancel()

	// Wait for worker pool to drain before shutting down HTTP.
	<-poolDone

	// Then drain connections with a deadline.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
