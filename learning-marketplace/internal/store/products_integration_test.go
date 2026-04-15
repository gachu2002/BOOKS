package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"learning-marketplace/internal/store"
	"learning-marketplace/internal/testutil"
)

func TestSearchPublishedProducts_ReturnsMatches(t *testing.T) {
	testDB := testutil.NewMigratedPostgres(t, "lm_test")
	s := store.New(testDB.DB)

	_, err := s.CreateProduct(context.Background(), store.ProductCreateParams{
		Slug:        "distributed-systems-lab",
		Title:       "Distributed Systems Lab",
		Description: "Learn distributed search and indexing",
		ProductType: "digital_download",
		PriceCents:  1000,
		Currency:    "USD",
		Published:   true,
	})
	require.NoError(t, err)

	_, err = s.CreateProduct(context.Background(), store.ProductCreateParams{
		Slug:        "private-draft-note",
		Title:       "Distributed Draft",
		Description: "Should not appear because it is unpublished",
		ProductType: "digital_download",
		PriceCents:  1000,
		Currency:    "USD",
		Published:   false,
	})
	require.NoError(t, err)

	results, err := s.SearchPublishedProducts(context.Background(), "distributed", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "distributed-systems-lab", results[0].Product.Slug)
	require.Greater(t, results[0].Rank, 0.0)
}

func TestListPublishedProductsByCursor_UsesStableKeysetPagination(t *testing.T) {
	testDB := testutil.NewMigratedPostgres(t, "lm_test")
	s := store.New(testDB.DB)

	first, err := s.CreateProduct(context.Background(), store.ProductCreateParams{Slug: "product-a", Title: "Product A", Description: "A", ProductType: "digital_download", PriceCents: 1000, Currency: "USD", Published: true})
	require.NoError(t, err)
	second, err := s.CreateProduct(context.Background(), store.ProductCreateParams{Slug: "product-b", Title: "Product B", Description: "B", ProductType: "digital_download", PriceCents: 1000, Currency: "USD", Published: true})
	require.NoError(t, err)
	third, err := s.CreateProduct(context.Background(), store.ProductCreateParams{Slug: "product-c", Title: "Product C", Description: "C", ProductType: "digital_download", PriceCents: 1000, Currency: "USD", Published: true})
	require.NoError(t, err)

	_, err = testDB.DB.ExecContext(context.Background(), `
UPDATE products
SET created_at = CASE slug
    WHEN 'product-a' THEN $1
    WHEN 'product-b' THEN $2
    WHEN 'product-c' THEN $3
END,
updated_at = CASE slug
    WHEN 'product-a' THEN $1
    WHEN 'product-b' THEN $2
    WHEN 'product-c' THEN $3
END
WHERE slug IN ('product-a', 'product-b', 'product-c')`,
		time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)

	pageOne, err := s.ListPublishedProductsByCursor(context.Background(), 2, nil)
	require.NoError(t, err)
	require.Len(t, pageOne, 2)
	require.Equal(t, third.ID, pageOne[0].ID)
	require.Equal(t, second.ID, pageOne[1].ID)

	cursor := &store.ProductCursor{CreatedAt: pageOne[1].CreatedAt, ID: pageOne[1].ID}
	pageTwo, err := s.ListPublishedProductsByCursor(context.Background(), 2, cursor)
	require.NoError(t, err)
	require.Len(t, pageTwo, 1)
	require.Equal(t, first.ID, pageTwo[0].ID)
}
