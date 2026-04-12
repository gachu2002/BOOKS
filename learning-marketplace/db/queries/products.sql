-- name: CreateProduct :one
INSERT INTO products (slug, title, description, product_type, price_cents, currency, published, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, slug, title, description, product_type, price_cents, currency, published, metadata, created_at, updated_at;

-- name: GetProduct :one
SELECT id, slug, title, description, product_type, price_cents, currency, published, metadata, created_at, updated_at
FROM products
WHERE id = $1;

-- name: ListProducts :many
SELECT id, slug, title, description, product_type, price_cents, currency, published, metadata, created_at, updated_at
FROM products
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateProduct :one
UPDATE products
SET slug = COALESCE($2, slug),
    title = COALESCE($3, title),
    description = COALESCE($4, description),
    product_type = COALESCE($5, product_type),
    price_cents = COALESCE($6, price_cents),
    currency = COALESCE($7, currency),
    published = COALESCE($8, published),
    metadata = CASE WHEN $9 THEN $10 ELSE metadata END,
    updated_at = now()
WHERE id = $1
RETURNING id, slug, title, description, product_type, price_cents, currency, published, metadata, created_at, updated_at;

-- name: DeleteProduct :execrows
DELETE FROM products WHERE id = $1;
