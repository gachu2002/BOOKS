package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"learning-marketplace/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Open(ctx context.Context, cfg config.PostgresConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}
