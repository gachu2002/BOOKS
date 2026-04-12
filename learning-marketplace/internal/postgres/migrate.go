package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	projectdb "learning-marketplace/db"
)

func Migrate(ctx context.Context, db *sql.DB) error {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return err
	}

	entries, err := fs.Glob(projectdb.Migrations, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}

	sort.Strings(entries)

	for _, path := range entries {
		name := strings.TrimPrefix(path, "migrations/")

		applied, err := migrationApplied(ctx, db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := fs.ReadFile(projectdb.Migrations, path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if err := applyMigration(ctx, db, name, string(content)); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	const query = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    name TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);`

	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	return nil
}

func migrationApplied(ctx context.Context, db *sql.DB, name string) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1);`

	var applied bool
	if err := db.QueryRowContext(ctx, query, name).Scan(&applied); err != nil {
		return false, fmt.Errorf("check migration %s: %w", name, err)
	}

	return applied, nil
}

func applyMigration(ctx context.Context, db *sql.DB, name string, sqlText string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, sqlText); err != nil {
		return fmt.Errorf("execute migration %s: %w", name, err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}

	return nil
}
