package store

import (
	"context"
	"fmt"
	"time"
)

type UserLibraryItem struct {
	OrderID         string    `json:"order_id"`
	UserID          string    `json:"user_id"`
	ProductID       string    `json:"product_id"`
	CohortID        *string   `json:"cohort_id,omitempty"`
	ProductSlug     string    `json:"product_slug"`
	ProductTitle    string    `json:"product_title"`
	CohortSlug      *string   `json:"cohort_slug,omitempty"`
	CohortTitle     *string   `json:"cohort_title,omitempty"`
	TotalCents      int       `json:"total_cents"`
	Currency        string    `json:"currency"`
	SourceEventID   string    `json:"source_event_id"`
	SourceEventType string    `json:"source_event_type"`
	ProjectedAt     time.Time `json:"projected_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UserLibraryProjectionStatus struct {
	PendingEvents        int        `json:"pending_events"`
	Stale                bool       `json:"stale"`
	LatestProjectedAt    *time.Time `json:"latest_projected_at,omitempty"`
	LatestSourceEventAt  *time.Time `json:"latest_source_event_at,omitempty"`
	OldestPendingEventAt *time.Time `json:"oldest_pending_event_at,omitempty"`
}

func (s *Store) ListUserLibrary(ctx context.Context, userID string, page Pagination) ([]UserLibraryItem, error) {
	const query = `
SELECT order_id, user_id, product_id, cohort_id, product_slug, product_title, cohort_slug, cohort_title, total_cents, currency, source_event_id, source_event_type, projected_at, updated_at
FROM user_library_projection
WHERE user_id = $1
ORDER BY projected_at DESC, order_id DESC
LIMIT $2 OFFSET $3`

	rows, err := s.db.QueryContext(ctx, query, userID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list user library: %w", err)
	}
	defer rows.Close()

	items := make([]UserLibraryItem, 0, page.Limit)
	for rows.Next() {
		var item UserLibraryItem
		if err := rows.Scan(
			&item.OrderID,
			&item.UserID,
			&item.ProductID,
			&item.CohortID,
			&item.ProductSlug,
			&item.ProductTitle,
			&item.CohortSlug,
			&item.CohortTitle,
			&item.TotalCents,
			&item.Currency,
			&item.SourceEventID,
			&item.SourceEventType,
			&item.ProjectedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user library item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user library: %w", err)
	}

	return items, nil
}

func (s *Store) GetUserLibraryProjectionStatus(ctx context.Context, userID string) (UserLibraryProjectionStatus, error) {
	const query = `
SELECT
    (
        SELECT COUNT(*)
        FROM outbox_events oe
        JOIN orders o ON o.id = oe.aggregate_id
        WHERE oe.aggregate_type = 'order'
          AND oe.event_type = 'order.paid'
          AND oe.published_at IS NULL
          AND o.user_id = $1
    ) AS pending_events,
    (
        SELECT MAX(projected_at)
        FROM user_library_projection
        WHERE user_id = $1
    ) AS latest_projected_at,
    (
        SELECT MAX(oe.created_at)
        FROM outbox_events oe
        JOIN orders o ON o.id = oe.aggregate_id
        WHERE oe.aggregate_type = 'order'
          AND oe.event_type = 'order.paid'
          AND o.user_id = $1
    ) AS latest_source_event_at,
    (
        SELECT MIN(oe.created_at)
        FROM outbox_events oe
        JOIN orders o ON o.id = oe.aggregate_id
        WHERE oe.aggregate_type = 'order'
          AND oe.event_type = 'order.paid'
          AND oe.published_at IS NULL
          AND o.user_id = $1
    ) AS oldest_pending_event_at`

	var status UserLibraryProjectionStatus
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&status.PendingEvents,
		&status.LatestProjectedAt,
		&status.LatestSourceEventAt,
		&status.OldestPendingEventAt,
	)
	if err != nil {
		return UserLibraryProjectionStatus{}, fmt.Errorf("get user library projection status: %w", err)
	}

	status.Stale = status.PendingEvents > 0

	return status, nil
}
