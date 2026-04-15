package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"learning-marketplace/internal/app"
	"learning-marketplace/internal/config"
	"learning-marketplace/internal/coordination"
	"learning-marketplace/internal/httpapi"
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

	var leaseStore *coordination.LeaseStore
	if cfg.Etcd.Enabled {
		leaseStore, err = coordination.Open(ctx, cfg.Etcd)
		if err != nil {
			slog.Error("failed to open etcd", "error", err)
			os.Exit(1)
		}
		defer func() {
			_ = leaseStore.Close()
		}()
	}

	var readerDB *sql.DB
	if cfg.PostgresReplica.Enabled {
		readerDB, err = postgres.Open(ctx, cfg.PostgresReplica.PostgresConfig)
		if err != nil {
			slog.Error("failed to open postgres replica", "error", err)
			os.Exit(1)
		}
		defer readerDB.Close()
	}

	userShardStores := make([]store.NamedStore, 0)
	for idx, shardCfg := range cfg.PostgresShards.Nodes() {
		shardDB, err := postgres.Open(ctx, shardCfg)
		if err != nil {
			slog.Error("failed to open postgres user shard", "index", idx, "error", err)
			os.Exit(1)
		}
		defer shardDB.Close()
		userShardStores = append(userShardStores, store.NamedStore{Name: cfg.PostgresShards.NameForIndex(idx), Store: store.New(shardDB)})
	}

	migrationCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := postgres.Migrate(migrationCtx, db); err != nil {
		slog.Error("failed to apply migrations", "error", err)
		os.Exit(1)
	}
	for idx, shardStore := range userShardStores {
		if err := postgres.Migrate(migrationCtx, shardStore.Store.DB()); err != nil {
			slog.Error("failed to apply shard migrations", "index", idx, "error", err)
			os.Exit(1)
		}
	}

	application := app.New(cfg, db, readerDB, userShardStores, leaseStore)
	router := httpapi.NewRouter(application)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting api",
			"http_addr", cfg.HTTPAddr,
			"env", cfg.AppEnv,
			"postgres_host", cfg.Postgres.Host,
			"postgres_port", cfg.Postgres.Port,
			"postgres_db", cfg.Postgres.DB,
			"replica_enabled", cfg.PostgresReplica.Enabled,
			"etcd_enabled", cfg.Etcd.Enabled,
		)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			slog.Error("http server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
		return
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		_ = server.Close()
	}

	if err := <-errCh; err != nil && err != http.ErrServerClosed {
		slog.Error("http server exited with error during shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("api stopped")
}
