package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"learning-marketplace/internal/config"
	"learning-marketplace/internal/postgres"
)

// ch4lab makes query-path and index behavior visible with seeded data and EXPLAIN output.
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

	if err := seedChapter4Lab(ctx, db); err != nil {
		slog.Error("failed to seed chapter 4 lab", "error", err)
		os.Exit(1)
	}

	cursor, err := publishedProductCursorAtOffset(ctx, db, 500)
	if err != nil {
		slog.Error("failed to load keyset cursor", "error", err)
		os.Exit(1)
	}

	plans := []struct {
		Name  string
		Query string
		Args  []any
	}{
		{
			Name: "Offset pagination on published products",
			Query: `
SELECT id, slug, title
FROM products
WHERE published = true
ORDER BY created_at DESC, id DESC
LIMIT 20 OFFSET 500`,
		},
		{
			Name: "Keyset pagination on published products",
			Query: `
SELECT id, slug, title
FROM products
WHERE published = true
  AND (created_at, id) < ($1, $2::uuid)
ORDER BY created_at DESC, id DESC
LIMIT 20`,
			Args: []any{cursor.CreatedAt, cursor.ID},
		},
		{
			Name: "Full-text search on published products",
			Query: `
SELECT id, slug, title
FROM products
WHERE published = true
  AND to_tsvector('simple', COALESCE(title, '') || ' ' || COALESCE(description, '')) @@ plainto_tsquery('simple', $1)
ORDER BY created_at DESC, id DESC
LIMIT 20`,
			Args: []any{"distributed"},
		},
		{
			Name: "Cohort listing by status and start time",
			Query: `
SELECT id, slug, status, starts_at
FROM cohorts
WHERE status = 'open'
ORDER BY starts_at ASC, created_at DESC
LIMIT 20 OFFSET 200`,
		},
	}

	for _, plan := range plans {
		fmt.Printf("\n== %s ==\n", plan.Name)
		if err := printExplain(ctx, db, plan.Query, plan.Args...); err != nil {
			slog.Error("failed to explain query", "name", plan.Name, "error", err)
			os.Exit(1)
		}
	}
}

type productCursor struct {
	CreatedAt time.Time
	ID        string
}

func seedChapter4Lab(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
INSERT INTO products (slug, title, description, product_type, price_cents, currency, published, metadata, created_at, updated_at)
SELECT
    format('lab-product-%s', gs),
    CASE WHEN gs % 10 = 0 THEN format('Distributed systems lab %s', gs) ELSE format('Backend study pack %s', gs) END,
    CASE WHEN gs % 15 = 0 THEN 'postgres indexing and distributed search practice' ELSE 'go backend practice and product catalog' END,
    CASE WHEN gs % 4 = 0 THEN 'live_cohort' ELSE 'digital_download' END,
    1000 + gs,
    'USD',
    gs % 5 <> 0,
    '{}'::jsonb,
    now() - make_interval(mins => gs),
    now() - make_interval(mins => gs)
FROM generate_series(1, 3000) gs
ON CONFLICT (slug) DO NOTHING`); err != nil {
		return fmt.Errorf("seed products: %w", err)
	}

	if _, err := db.ExecContext(ctx, `
INSERT INTO cohorts (
    product_id,
    slug,
    title,
    capacity,
    status,
    starts_at,
    ends_at,
    enrollment_opens_at,
    enrollment_closes_at,
    created_at,
    updated_at
)
SELECT
    p.id,
    format('lab-cohort-%s', gs),
    format('Lab Cohort %s', gs),
    30 + (gs % 20),
    CASE WHEN gs % 3 = 0 THEN 'open' WHEN gs % 3 = 1 THEN 'draft' ELSE 'closed' END,
    now() + make_interval(days => gs),
    now() + make_interval(days => gs + 7),
    now() + make_interval(days => gs - 30),
    now() + make_interval(days => gs - 1),
    now() - make_interval(days => gs),
    now() - make_interval(days => gs)
FROM generate_series(1, 750) gs
JOIN products p ON p.slug = format('lab-product-%s', gs * 4)
ON CONFLICT (slug) DO NOTHING`); err != nil {
		return fmt.Errorf("seed cohorts: %w", err)
	}

	return nil
}

func publishedProductCursorAtOffset(ctx context.Context, db *sql.DB, offset int) (productCursor, error) {
	const query = `
SELECT created_at, id
FROM products
WHERE published = true
ORDER BY created_at DESC, id DESC
LIMIT 1 OFFSET $1`

	var cursor productCursor
	if err := db.QueryRowContext(ctx, query, offset).Scan(&cursor.CreatedAt, &cursor.ID); err != nil {
		return productCursor{}, fmt.Errorf("fetch cursor at offset %d: %w", offset, err)
	}

	return cursor, nil
}

func printExplain(ctx context.Context, db *sql.DB, query string, args ...any) error {
	rows, err := db.QueryContext(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+query, args...)
	if err != nil {
		return fmt.Errorf("run explain: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return fmt.Errorf("scan explain line: %w", err)
		}
		fmt.Println(strings.TrimSpace(line))
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate explain lines: %w", err)
	}

	return nil
}
