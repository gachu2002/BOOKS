package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Reporter struct {
	db *sql.DB
}

type RebuildMetadata struct {
	RebuiltBy     string
	FencingToken  int64
	RebuiltAtTime time.Time
}

// NewReporter builds deterministic batch tables from the operational store.
func NewReporter(db *sql.DB) *Reporter {
	return &Reporter{db: db}
}

type DailyRevenueRow struct {
	ReportDate        time.Time `json:"report_date"`
	Currency          string    `json:"currency"`
	OrdersCount       int       `json:"orders_count"`
	GrossRevenueCents int64     `json:"gross_revenue_cents"`
	DiscountCents     int64     `json:"discount_cents"`
	NetRevenueCents   int64     `json:"net_revenue_cents"`
	RebuiltBy         string    `json:"rebuilt_by"`
	FencingToken      int64     `json:"rebuild_fencing_token"`
	RebuiltAt         time.Time `json:"rebuilt_at"`
}

type CohortFillRow struct {
	CohortID        string    `json:"cohort_id"`
	ProductID       string    `json:"product_id"`
	CohortSlug      string    `json:"cohort_slug"`
	CohortTitle     string    `json:"cohort_title"`
	Capacity        int       `json:"capacity"`
	SoldSeats       int       `json:"sold_seats"`
	FillRatePercent float64   `json:"fill_rate_percent"`
	RevenueCents    int64     `json:"revenue_cents"`
	StartsAt        time.Time `json:"starts_at"`
	RebuiltBy       string    `json:"rebuilt_by"`
	FencingToken    int64     `json:"rebuild_fencing_token"`
	RebuiltAt       time.Time `json:"rebuilt_at"`
}

// Rebuild recreates analytics tables from source-of-truth operational rows.
func (r *Reporter) Rebuild(ctx context.Context, metadata RebuildMetadata) error {
	if metadata.RebuiltBy == "" {
		metadata.RebuiltBy = "batch-reports"
	}
	if metadata.RebuiltAtTime.IsZero() {
		metadata.RebuiltAtTime = time.Now().UTC()
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin analytics rebuild: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `TRUNCATE TABLE analytics_daily_revenue, analytics_cohort_fill`); err != nil {
		return fmt.Errorf("truncate analytics tables: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO analytics_daily_revenue (
    report_date,
    currency,
    orders_count,
    gross_revenue_cents,
    discount_cents,
    net_revenue_cents,
    rebuilt_by,
    rebuild_fencing_token,
    rebuilt_at
)
SELECT
    DATE(o.placed_at AT TIME ZONE 'UTC') AS report_date,
    o.currency,
    COUNT(*) AS orders_count,
    COALESCE(SUM(o.subtotal_cents), 0) AS gross_revenue_cents,
    COALESCE(SUM(o.discount_cents), 0) AS discount_cents,
    COALESCE(SUM(o.total_cents), 0) AS net_revenue_cents,
    $1 AS rebuilt_by,
    $2 AS rebuild_fencing_token,
    $3 AS rebuilt_at
FROM orders o
WHERE o.status = 'paid'
  AND o.placed_at IS NOT NULL
GROUP BY DATE(o.placed_at AT TIME ZONE 'UTC'), o.currency`, metadata.RebuiltBy, metadata.FencingToken, metadata.RebuiltAtTime); err != nil {
		return fmt.Errorf("rebuild daily revenue report: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO analytics_cohort_fill (
    cohort_id,
    product_id,
    cohort_slug,
    cohort_title,
    capacity,
    sold_seats,
    fill_rate_percent,
    revenue_cents,
    starts_at,
    rebuilt_by,
    rebuild_fencing_token,
    rebuilt_at
)
SELECT
    c.id,
    c.product_id,
    c.slug,
    c.title,
    c.capacity,
    COALESCE(SUM(CASE WHEN o.status = 'paid' THEN oi.quantity ELSE 0 END), 0) AS sold_seats,
    ROUND(
        COALESCE(SUM(CASE WHEN o.status = 'paid' THEN oi.quantity ELSE 0 END), 0)::numeric * 100 / c.capacity,
        2
    ) AS fill_rate_percent,
    COALESCE(SUM(CASE WHEN o.status = 'paid' THEN oi.line_total_cents ELSE 0 END), 0) AS revenue_cents,
    c.starts_at,
    $1 AS rebuilt_by,
    $2 AS rebuild_fencing_token,
    $3 AS rebuilt_at
FROM cohorts c
LEFT JOIN order_items oi ON oi.cohort_id = c.id
LEFT JOIN orders o ON o.id = oi.order_id
GROUP BY c.id, c.product_id, c.slug, c.title, c.capacity, c.starts_at`, metadata.RebuiltBy, metadata.FencingToken, metadata.RebuiltAtTime); err != nil {
		return fmt.Errorf("rebuild cohort fill report: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit analytics rebuild: %w", err)
	}

	return nil
}

func (r *Reporter) ListDailyRevenue(ctx context.Context, limit, offset int) ([]DailyRevenueRow, error) {
	const query = `
SELECT report_date, currency, orders_count, gross_revenue_cents, discount_cents, net_revenue_cents, rebuilt_by, rebuild_fencing_token, rebuilt_at
FROM analytics_daily_revenue
ORDER BY report_date DESC, currency ASC
LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list daily revenue: %w", err)
	}
	defer rows.Close()

	items := make([]DailyRevenueRow, 0, limit)
	for rows.Next() {
		var item DailyRevenueRow
		if err := rows.Scan(&item.ReportDate, &item.Currency, &item.OrdersCount, &item.GrossRevenueCents, &item.DiscountCents, &item.NetRevenueCents, &item.RebuiltBy, &item.FencingToken, &item.RebuiltAt); err != nil {
			return nil, fmt.Errorf("scan daily revenue row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daily revenue rows: %w", err)
	}

	return items, nil
}

func (r *Reporter) ListCohortFill(ctx context.Context, limit, offset int) ([]CohortFillRow, error) {
	const query = `
SELECT cohort_id, product_id, cohort_slug, cohort_title, capacity, sold_seats, fill_rate_percent, revenue_cents, starts_at, rebuilt_by, rebuild_fencing_token, rebuilt_at
FROM analytics_cohort_fill
ORDER BY starts_at DESC, cohort_slug ASC
LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list cohort fill: %w", err)
	}
	defer rows.Close()

	items := make([]CohortFillRow, 0, limit)
	for rows.Next() {
		var item CohortFillRow
		if err := rows.Scan(&item.CohortID, &item.ProductID, &item.CohortSlug, &item.CohortTitle, &item.Capacity, &item.SoldSeats, &item.FillRatePercent, &item.RevenueCents, &item.StartsAt, &item.RebuiltBy, &item.FencingToken, &item.RebuiltAt); err != nil {
			return nil, fmt.Errorf("scan cohort fill row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cohort fill rows: %w", err)
	}

	return items, nil
}
