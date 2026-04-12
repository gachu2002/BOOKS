package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (s *Store) CreatePromoCode(ctx context.Context, params PromoCodeCreateParams) (PromoCode, error) {
	const query = `
INSERT INTO promo_codes (code, discount_type, discount_value, max_redemptions, starts_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, code, discount_type, discount_value, max_redemptions, starts_at, expires_at, created_at, updated_at`

	var promo PromoCode
	err := s.db.QueryRowContext(ctx, query,
		params.Code,
		params.DiscountType,
		params.DiscountValue,
		params.MaxRedemptions,
		params.StartsAt,
		params.ExpiresAt,
	).Scan(
		&promo.ID,
		&promo.Code,
		&promo.DiscountType,
		&promo.DiscountValue,
		&promo.MaxRedemptions,
		&promo.StartsAt,
		&promo.ExpiresAt,
		&promo.CreatedAt,
		&promo.UpdatedAt,
	)
	if err != nil {
		return PromoCode{}, fmt.Errorf("create promo code: %w", err)
	}

	return promo, nil
}

func (s *Store) GetPromoCode(ctx context.Context, id string) (PromoCode, error) {
	const query = `
SELECT id, code, discount_type, discount_value, max_redemptions, starts_at, expires_at, created_at, updated_at
FROM promo_codes
WHERE id = $1`

	var promo PromoCode
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&promo.ID,
		&promo.Code,
		&promo.DiscountType,
		&promo.DiscountValue,
		&promo.MaxRedemptions,
		&promo.StartsAt,
		&promo.ExpiresAt,
		&promo.CreatedAt,
		&promo.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PromoCode{}, ErrNotFound
		}
		return PromoCode{}, fmt.Errorf("get promo code: %w", err)
	}

	return promo, nil
}

func (s *Store) ListPromoCodes(ctx context.Context, page Pagination) ([]PromoCode, error) {
	const query = `
SELECT id, code, discount_type, discount_value, max_redemptions, starts_at, expires_at, created_at, updated_at
FROM promo_codes
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`

	rows, err := s.db.QueryContext(ctx, query, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list promo codes: %w", err)
	}
	defer rows.Close()

	promos := make([]PromoCode, 0, page.Limit)
	for rows.Next() {
		var promo PromoCode
		if err := rows.Scan(
			&promo.ID,
			&promo.Code,
			&promo.DiscountType,
			&promo.DiscountValue,
			&promo.MaxRedemptions,
			&promo.StartsAt,
			&promo.ExpiresAt,
			&promo.CreatedAt,
			&promo.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan promo code: %w", err)
		}
		promos = append(promos, promo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate promo codes: %w", err)
	}

	return promos, nil
}

func (s *Store) UpdatePromoCode(ctx context.Context, id string, params PromoCodeUpdateParams) (PromoCode, error) {
	const query = `
UPDATE promo_codes
SET code = COALESCE($2, code),
    discount_type = COALESCE($3, discount_type),
    discount_value = COALESCE($4, discount_value),
    max_redemptions = CASE WHEN $5 THEN $6 ELSE max_redemptions END,
    starts_at = CASE WHEN $7 THEN $8 ELSE starts_at END,
    expires_at = CASE WHEN $9 THEN $10 ELSE expires_at END,
    updated_at = now()
WHERE id = $1
RETURNING id, code, discount_type, discount_value, max_redemptions, starts_at, expires_at, created_at, updated_at`

	var promo PromoCode
	err := s.db.QueryRowContext(ctx, query,
		id,
		params.Code,
		params.DiscountType,
		params.DiscountValue,
		params.UpdateMaxRedemptions,
		nullIntArg(params.MaxRedemptions),
		params.UpdateStartsAt,
		nullTimeArg(params.StartsAt),
		params.UpdateExpiresAt,
		nullTimeArg(params.ExpiresAt),
	).Scan(
		&promo.ID,
		&promo.Code,
		&promo.DiscountType,
		&promo.DiscountValue,
		&promo.MaxRedemptions,
		&promo.StartsAt,
		&promo.ExpiresAt,
		&promo.CreatedAt,
		&promo.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PromoCode{}, ErrNotFound
		}
		return PromoCode{}, fmt.Errorf("update promo code: %w", err)
	}

	return promo, nil
}

func (s *Store) DeletePromoCode(ctx context.Context, id string) error {
	const query = `DELETE FROM promo_codes WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete promo code: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete promo code rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func nullIntArg(value sql.NullInt32) any {
	if !value.Valid {
		return nil
	}

	return value.Int32
}
