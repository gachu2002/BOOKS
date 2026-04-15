package projector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"learning-marketplace/internal/store"
)

type UserLibraryProjector struct {
	db *sql.DB
}

// NewUserLibraryProjector builds the first outbox-driven read model for purchased content.
func NewUserLibraryProjector(db *sql.DB) *UserLibraryProjector {
	return &UserLibraryProjector{db: db}
}

type outboxEvent struct {
	ID      string
	Type    string
	Payload json.RawMessage
}

type orderPaidPayload struct {
	SchemaVersion int    `json:"schema_version"`
	OrderID       string `json:"order_id"`
	UserID        string `json:"user_id"`
	ProductID     string `json:"product_id"`
	CohortID      string `json:"cohort_id"`
	PaymentID     string `json:"payment_id"`
	EntitlementID string `json:"entitlement_id"`
	TotalCents    int    `json:"total_cents"`
	Currency      string `json:"currency"`
}

// ProcessPending advances the derived view by consuming unpublished outbox events.
func (p *UserLibraryProjector) ProcessPending(ctx context.Context, limit int) (int, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin projector transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	events, err := loadPendingOutboxEvents(ctx, tx, limit)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, event := range events {
		if err := p.applyEvent(ctx, tx, event); err != nil {
			return 0, err
		}
		if err := markOutboxEventPublished(ctx, tx, event.ID); err != nil {
			return 0, err
		}
		processed++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit projector transaction: %w", err)
	}

	return processed, nil
}

// Rebuild drops and recreates the derived state from the full event log.
func (p *UserLibraryProjector) Rebuild(ctx context.Context) (int, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin projector rebuild: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `TRUNCATE TABLE user_library_projection`); err != nil {
		return 0, fmt.Errorf("truncate user_library_projection: %w", err)
	}

	events, err := loadAllProjectableEvents(ctx, tx)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, event := range events {
		if err := p.applyEvent(ctx, tx, event); err != nil {
			return 0, err
		}
		processed++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit projector rebuild: %w", err)
	}

	return processed, nil
}

// applyEvent keeps projection logic explicit so drift and replay are easy to inspect.
func (p *UserLibraryProjector) applyEvent(ctx context.Context, tx *sql.Tx, event outboxEvent) error {
	switch event.Type {
	case "order.paid":
		var payload orderPaidPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode outbox event %s: %w", event.ID, err)
		}
		if payload.SchemaVersion == 0 {
			payload.SchemaVersion = 1
		}
		if payload.SchemaVersion != 1 {
			return fmt.Errorf("decode outbox event %s: unsupported order.paid schema_version %d", event.ID, payload.SchemaVersion)
		}

		product, cohort, err := loadProjectionSourceData(ctx, tx, payload.ProductID, payload.CohortID)
		if err != nil {
			return err
		}

		if err := upsertUserLibraryProjection(ctx, tx, event.ID, payload, product, cohort); err != nil {
			return err
		}

		return nil
	default:
		return nil
	}
}

func loadPendingOutboxEvents(ctx context.Context, tx *sql.Tx, limit int) ([]outboxEvent, error) {
	const query = `
SELECT id, event_type, payload
FROM outbox_events
WHERE published_at IS NULL
ORDER BY created_at ASC, id ASC
LIMIT $1
FOR UPDATE SKIP LOCKED`

	rows, err := tx.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("load pending outbox events: %w", err)
	}
	defer rows.Close()

	events := make([]outboxEvent, 0, limit)
	for rows.Next() {
		var event outboxEvent
		if err := rows.Scan(&event.ID, &event.Type, &event.Payload); err != nil {
			return nil, fmt.Errorf("scan pending outbox event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending outbox events: %w", err)
	}

	return events, nil
}

func loadAllProjectableEvents(ctx context.Context, tx *sql.Tx) ([]outboxEvent, error) {
	const query = `
SELECT id, event_type, payload
FROM outbox_events
WHERE event_type = 'order.paid'
ORDER BY created_at ASC, id ASC`

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("load all projectable events: %w", err)
	}
	defer rows.Close()

	events := make([]outboxEvent, 0)
	for rows.Next() {
		var event outboxEvent
		if err := rows.Scan(&event.ID, &event.Type, &event.Payload); err != nil {
			return nil, fmt.Errorf("scan projectable outbox event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projectable outbox events: %w", err)
	}

	return events, nil
}

func markOutboxEventPublished(ctx context.Context, tx *sql.Tx, eventID string) error {
	const query = `UPDATE outbox_events SET published_at = now() WHERE id = $1`
	if _, err := tx.ExecContext(ctx, query, eventID); err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}

	return nil
}

func loadProjectionSourceData(ctx context.Context, tx *sql.Tx, productID string, cohortID string) (store.Product, *store.Cohort, error) {
	const productQuery = `
SELECT id, slug, title, description, product_type, price_cents, currency, published, metadata, created_at, updated_at
FROM products WHERE id = $1`

	var product store.Product
	if err := tx.QueryRowContext(ctx, productQuery, productID).Scan(
		&product.ID,
		&product.Slug,
		&product.Title,
		&product.Description,
		&product.ProductType,
		&product.PriceCents,
		&product.Currency,
		&product.Published,
		&product.Metadata,
		&product.CreatedAt,
		&product.UpdatedAt,
	); err != nil {
		return store.Product{}, nil, fmt.Errorf("load product for projection: %w", err)
	}

	if cohortID == "" {
		return product, nil, nil
	}

	const cohortQuery = `
SELECT id, product_id, slug, title, capacity, status, starts_at, ends_at, enrollment_opens_at, enrollment_closes_at, created_at, updated_at
FROM cohorts WHERE id = $1`

	var cohort store.Cohort
	if err := tx.QueryRowContext(ctx, cohortQuery, cohortID).Scan(
		&cohort.ID,
		&cohort.ProductID,
		&cohort.Slug,
		&cohort.Title,
		&cohort.Capacity,
		&cohort.Status,
		&cohort.StartsAt,
		&cohort.EndsAt,
		&cohort.EnrollmentOpensAt,
		&cohort.EnrollmentClosesAt,
		&cohort.CreatedAt,
		&cohort.UpdatedAt,
	); err != nil {
		return store.Product{}, nil, fmt.Errorf("load cohort for projection: %w", err)
	}

	return product, &cohort, nil
}

func upsertUserLibraryProjection(ctx context.Context, tx *sql.Tx, sourceEventID string, payload orderPaidPayload, product store.Product, cohort *store.Cohort) error {
	const query = `
INSERT INTO user_library_projection (
    order_id,
    user_id,
    product_id,
    cohort_id,
    product_slug,
    product_title,
    cohort_slug,
    cohort_title,
    total_cents,
    currency,
    source_event_id,
    source_event_type,
    projected_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'order.paid', now(), now())
ON CONFLICT (order_id) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    product_id = EXCLUDED.product_id,
    cohort_id = EXCLUDED.cohort_id,
    product_slug = EXCLUDED.product_slug,
    product_title = EXCLUDED.product_title,
    cohort_slug = EXCLUDED.cohort_slug,
    cohort_title = EXCLUDED.cohort_title,
    total_cents = EXCLUDED.total_cents,
    currency = EXCLUDED.currency,
    source_event_id = EXCLUDED.source_event_id,
    source_event_type = EXCLUDED.source_event_type,
    updated_at = now()`

	var cohortID any
	var cohortSlug any
	var cohortTitle any
	if cohort != nil {
		cohortID = cohort.ID
		cohortSlug = cohort.Slug
		cohortTitle = cohort.Title
	}

	if _, err := tx.ExecContext(ctx, query,
		payload.OrderID,
		payload.UserID,
		payload.ProductID,
		cohortID,
		product.Slug,
		product.Title,
		cohortSlug,
		cohortTitle,
		payload.TotalCents,
		payload.Currency,
		sourceEventID,
	); err != nil {
		return fmt.Errorf("upsert user library projection: %w", err)
	}

	return nil
}
