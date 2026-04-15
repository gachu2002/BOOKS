package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func (s *Store) CreateProduct(ctx context.Context, params ProductCreateParams) (Product, error) {
	const query = `
INSERT INTO products (slug, title, description, product_type, price_cents, currency, published, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, slug, title, description, product_type, price_cents, currency, published, metadata, created_at, updated_at`

	var product Product
	err := s.db.QueryRowContext(ctx, query,
		params.Slug,
		params.Title,
		params.Description,
		params.ProductType,
		params.PriceCents,
		params.Currency,
		params.Published,
		normalizeJSON(params.Metadata),
	).Scan(
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
	)
	if err != nil {
		return Product{}, fmt.Errorf("create product: %w", err)
	}

	return product, nil
}

func (s *Store) GetProduct(ctx context.Context, id string) (Product, error) {
	const query = `
SELECT id, slug, title, description, product_type, price_cents, currency, published, metadata, created_at, updated_at
FROM products
WHERE id = $1`

	var product Product
	err := s.db.QueryRowContext(ctx, query, id).Scan(
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
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Product{}, ErrNotFound
		}
		return Product{}, fmt.Errorf("get product: %w", err)
	}

	return product, nil
}

func (s *Store) ListProducts(ctx context.Context, page Pagination) ([]Product, error) {
	const query = `
SELECT id, slug, title, description, product_type, price_cents, currency, published, metadata, created_at, updated_at
FROM products
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2`

	rows, err := s.db.QueryContext(ctx, query, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	products := make([]Product, 0, page.Limit)
	for rows.Next() {
		var product Product
		if err := rows.Scan(
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
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}

	return products, nil
}

// ListPublishedProductsByCursor demonstrates keyset pagination for stable catalog reads.
func (s *Store) ListPublishedProductsByCursor(ctx context.Context, limit int, after *ProductCursor) ([]Product, error) {
	const query = `
SELECT id, slug, title, description, product_type, price_cents, currency, published, metadata, created_at, updated_at
FROM products
WHERE published = true
  AND ($2::timestamptz IS NULL OR (created_at, id) < ($2, $3::uuid))
ORDER BY created_at DESC, id DESC
LIMIT $1`

	var afterCreatedAt any
	var afterID any
	if after != nil {
		afterCreatedAt = after.CreatedAt
		afterID = after.ID
	}

	rows, err := s.db.QueryContext(ctx, query, limit, afterCreatedAt, afterID)
	if err != nil {
		return nil, fmt.Errorf("list published products by cursor: %w", err)
	}
	defer rows.Close()

	products := make([]Product, 0, limit)
	for rows.Next() {
		var product Product
		if err := rows.Scan(
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
			return nil, fmt.Errorf("scan published product: %w", err)
		}
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate published products: %w", err)
	}

	return products, nil
}

// SearchPublishedProducts demonstrates full-text retrieval over published catalog entries.
func (s *Store) SearchPublishedProducts(ctx context.Context, searchQuery string, limit int) ([]ProductSearchResult, error) {
	const query = `
SELECT
    id,
    slug,
    title,
    description,
    product_type,
    price_cents,
    currency,
    published,
    metadata,
    created_at,
    updated_at,
    ts_rank(
        to_tsvector('simple', COALESCE(title, '') || ' ' || COALESCE(description, '')),
        plainto_tsquery('simple', $1)
    ) AS rank
FROM products
WHERE published = true
  AND to_tsvector('simple', COALESCE(title, '') || ' ' || COALESCE(description, '')) @@ plainto_tsquery('simple', $1)
ORDER BY rank DESC, created_at DESC, id DESC
LIMIT $2`

	trimmed := strings.TrimSpace(searchQuery)
	if trimmed == "" {
		return []ProductSearchResult{}, nil
	}

	rows, err := s.db.QueryContext(ctx, query, trimmed, limit)
	if err != nil {
		return nil, fmt.Errorf("search published products: %w", err)
	}
	defer rows.Close()

	results := make([]ProductSearchResult, 0, limit)
	for rows.Next() {
		var item ProductSearchResult
		if err := rows.Scan(
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
			&item.Rank,
		); err != nil {
			return nil, fmt.Errorf("scan product search result: %w", err)
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product search results: %w", err)
	}

	return results, nil
}

func (s *Store) UpdateProduct(ctx context.Context, id string, params ProductUpdateParams) (Product, error) {
	const query = `
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
RETURNING id, slug, title, description, product_type, price_cents, currency, published, metadata, created_at, updated_at`

	var product Product
	err := s.db.QueryRowContext(ctx, query,
		id,
		params.Slug,
		params.Title,
		params.Description,
		params.ProductType,
		params.PriceCents,
		params.Currency,
		params.Published,
		params.UpdateMeta,
		normalizeJSON(params.Metadata),
	).Scan(
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
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Product{}, ErrNotFound
		}
		return Product{}, fmt.Errorf("update product: %w", err)
	}

	return product, nil
}

func (s *Store) DeleteProduct(ctx context.Context, id string) error {
	const query = `DELETE FROM products WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete product rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func normalizeJSON(value json.RawMessage) []byte {
	if len(value) == 0 {
		return []byte("{}")
	}

	return value
}
