-- name: CreatePromoCode :one
INSERT INTO promo_codes (code, discount_type, discount_value, max_redemptions, starts_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, code, discount_type, discount_value, max_redemptions, starts_at, expires_at, created_at, updated_at;

-- name: GetPromoCode :one
SELECT id, code, discount_type, discount_value, max_redemptions, starts_at, expires_at, created_at, updated_at
FROM promo_codes
WHERE id = $1;

-- name: ListPromoCodes :many
SELECT id, code, discount_type, discount_value, max_redemptions, starts_at, expires_at, created_at, updated_at
FROM promo_codes
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdatePromoCode :one
UPDATE promo_codes
SET code = COALESCE($2, code),
    discount_type = COALESCE($3, discount_type),
    discount_value = COALESCE($4, discount_value),
    max_redemptions = CASE WHEN $5 THEN $6 ELSE max_redemptions END,
    starts_at = CASE WHEN $7 THEN $8 ELSE starts_at END,
    expires_at = CASE WHEN $9 THEN $10 ELSE expires_at END,
    updated_at = now()
WHERE id = $1
RETURNING id, code, discount_type, discount_value, max_redemptions, starts_at, expires_at, created_at, updated_at;

-- name: DeletePromoCode :execrows
DELETE FROM promo_codes WHERE id = $1;
