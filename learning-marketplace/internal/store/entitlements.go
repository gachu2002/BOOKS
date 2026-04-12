package store

import (
	"context"
	"fmt"
)

func (s *Store) ListEntitlementsByUser(ctx context.Context, userID string, page Pagination) ([]EntitlementWithProduct, error) {
	const query = `
SELECT
    e.id,
    e.user_id,
    e.product_id,
    e.order_id,
    e.cohort_id,
    e.status,
    e.granted_at,
    e.revoked_at,
    e.created_at,
    e.updated_at,
    p.id,
    p.slug,
    p.title,
    p.description,
    p.product_type,
    p.price_cents,
    p.currency,
    p.published,
    p.metadata,
    p.created_at,
    p.updated_at
FROM entitlements e
JOIN products p ON p.id = e.product_id
WHERE e.user_id = $1
ORDER BY e.created_at DESC
LIMIT $2 OFFSET $3`

	rows, err := s.db.QueryContext(ctx, query, userID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list entitlements by user: %w", err)
	}
	defer rows.Close()

	items := make([]EntitlementWithProduct, 0, page.Limit)
	for rows.Next() {
		var item EntitlementWithProduct
		if err := rows.Scan(
			&item.Entitlement.ID,
			&item.Entitlement.UserID,
			&item.Entitlement.ProductID,
			&item.Entitlement.OrderID,
			&item.Entitlement.CohortID,
			&item.Entitlement.Status,
			&item.Entitlement.GrantedAt,
			&item.Entitlement.RevokedAt,
			&item.Entitlement.CreatedAt,
			&item.Entitlement.UpdatedAt,
			&item.Product.ID,
			&item.Product.Slug,
			&item.Product.Title,
			&item.Product.Description,
			&item.Product.ProductType,
			&item.Product.PriceCents,
			&item.Product.Currency,
			&item.Product.Published,
			&item.Product.Metadata,
			&item.Product.CreatedAt,
			&item.Product.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan entitlement: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entitlements: %w", err)
	}

	return items, nil
}
