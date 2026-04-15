package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"learning-marketplace/internal/analytics"
	"learning-marketplace/internal/config"
	"learning-marketplace/internal/coordination"
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

	metadata := analytics.RebuildMetadata{RebuiltBy: batchReportsHolder()}
	if cfg.Etcd.Enabled {
		leaseStore, err := coordination.Open(ctx, cfg.Etcd)
		if err != nil {
			slog.Error("failed to open lease store", "error", err)
			os.Exit(1)
		}
		defer leaseStore.Close()

		grant, err := leaseStore.Acquire(ctx, "batch-reports", metadata.RebuiltBy, 120)
		if err != nil {
			if err == coordination.ErrLeaseAlreadyHeld {
				slog.Info("batch reports rebuild skipped", "reason", "lease already held")
				return
			}
			slog.Error("failed to acquire batch reports lease", "error", err)
			os.Exit(1)
		}
		defer func() {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := leaseStore.Release(releaseCtx, grant.LeaseID); err != nil {
				slog.Warn("failed to release batch reports lease", "lease_id", grant.LeaseID, "error", err)
			}
		}()

		if err := leaseStore.KeepAlive(ctx, grant.LeaseID); err != nil {
			slog.Error("failed to keep batch reports lease alive", "error", err)
			os.Exit(1)
		}

		metadata.FencingToken = grant.FencingToken
		slog.Info("acquired batch reports lease", "holder", metadata.RebuiltBy, "fencing_token", grant.FencingToken, "ttl_seconds", grant.TTLSeconds)
	}

	reporter := analytics.NewReporter(db)
	if err := reporter.Rebuild(ctx, metadata); err != nil {
		slog.Error("failed to rebuild batch reports", "error", err)
		os.Exit(1)
	}

	slog.Info("batch reports rebuilt", "rebuilt_by", metadata.RebuiltBy, "fencing_token", metadata.FencingToken)
}

func batchReportsHolder() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown-host"
	}

	return hostname + ":" + strconv.Itoa(os.Getpid())
}
