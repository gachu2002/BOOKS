package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"learning-marketplace/internal/analytics"
	"learning-marketplace/internal/config"
	"learning-marketplace/internal/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(ctx, cfg.Postgres)
	if err != nil {
		slog.Error("failed to open postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	migrationCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := postgres.Migrate(migrationCtx, db); err != nil {
		slog.Error("failed to apply migrations", "error", err)
		os.Exit(1)
	}

	reporter := analytics.NewReporter(db)
	if err := reporter.Rebuild(ctx); err != nil {
		slog.Error("failed to rebuild batch reports", "error", err)
		os.Exit(1)
	}

	slog.Info("batch reports rebuilt")
}
