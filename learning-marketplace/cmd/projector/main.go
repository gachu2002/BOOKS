package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"learning-marketplace/internal/config"
	"learning-marketplace/internal/postgres"
	"learning-marketplace/internal/projector"
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

	p := projector.NewUserLibraryProjector(db)
	mode := os.Getenv("PROJECTOR_MODE")
	if mode == "" {
		mode = "poll"
	}

	slog.Info("starting projector", "mode", mode, "batch_size", cfg.Projector.BatchSize, "poll_interval", cfg.Projector.PollInterval.String())

	switch mode {
	case "rebuild":
		processed, err := p.Rebuild(ctx)
		if err != nil {
			slog.Error("projector rebuild failed", "error", err)
			os.Exit(1)
		}
		slog.Info("projector rebuild complete", "processed_events", processed)
	case "once":
		processed, err := p.ProcessPending(ctx, cfg.Projector.BatchSize)
		if err != nil {
			slog.Error("projector run failed", "error", err)
			os.Exit(1)
		}
		slog.Info("projector run complete", "processed_events", processed)
	default:
		pollTicker := time.NewTicker(cfg.Projector.PollInterval)
		defer pollTicker.Stop()

		for {
			processed, err := p.ProcessPending(ctx, cfg.Projector.BatchSize)
			if err != nil {
				if ctx.Err() != nil {
					break
				}
				slog.Error("projector polling failed", "error", err)
				os.Exit(1)
			}
			if processed > 0 {
				slog.Info("projector applied batch", "processed_events", processed)
			}

			select {
			case <-ctx.Done():
				slog.Info("projector stopping")
				return
			case <-pollTicker.C:
			}
		}
	}
}
