package projector_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"learning-marketplace/internal/projector"
	"learning-marketplace/internal/store"
	"learning-marketplace/internal/testutil"
)

func TestUserLibraryProjector_ProcessPendingAndRebuild(t *testing.T) {
	testDB := testutil.NewMigratedPostgres(t, "lm_projector_test")
	s := store.New(testDB.DB)
	p := projector.NewUserLibraryProjector(testDB.DB)

	user := createProjectorUserFixture(t, s, "projection@example.com")
	product := createProjectorProductFixture(t, s, "projection-product")
	cohort := createProjectorCohortFixture(t, s, product.ID, "projection-cohort", 2)

	_, err := s.CheckoutCohort(context.Background(), store.CheckoutParams{
		UserID:          user.ID,
		CohortID:        cohort.ID,
		PaymentProvider: "demo",
		IdempotencyKey:  "projection-001",
	})
	require.NoError(t, err)

	require.Equal(t, 0, countProjectionRows(t, testDB.DB))

	processed, err := p.ProcessPending(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 1, processed)
	require.Equal(t, 1, countProjectionRows(t, testDB.DB))

	items, err := s.ListUserLibrary(context.Background(), user.ID, store.Pagination{Limit: 10, Offset: 0})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, product.Title, items[0].ProductTitle)
	require.NotEmpty(t, items[0].SourceEventID)

	processed, err = p.ProcessPending(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 0, processed)
	require.Equal(t, 1, countProjectionRows(t, testDB.DB))

	require.NoError(t, truncateProjection(t, testDB.DB))
	require.Equal(t, 0, countProjectionRows(t, testDB.DB))

	replayed, err := p.Rebuild(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, replayed)
	require.Equal(t, 1, countProjectionRows(t, testDB.DB))
}

func createProjectorUserFixture(t *testing.T, s *store.Store, email string) store.User {
	t.Helper()
	user, err := s.CreateUser(context.Background(), store.UserCreateParams{Email: email, FullName: email, Role: "student"})
	require.NoError(t, err)
	return user
}

func createProjectorProductFixture(t *testing.T, s *store.Store, slug string) store.Product {
	t.Helper()
	product, err := s.CreateProduct(context.Background(), store.ProductCreateParams{Slug: slug, Title: slug, Description: "projection", ProductType: "live_cohort", PriceCents: 10000, Currency: "USD", Published: true})
	require.NoError(t, err)
	return product
}

func createProjectorCohortFixture(t *testing.T, s *store.Store, productID, slug string, capacity int) store.Cohort {
	t.Helper()
	startsAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	endsAt := startsAt.Add(7 * 24 * time.Hour)
	cohort, err := s.CreateCohort(context.Background(), store.CohortCreateParams{ProductID: productID, Slug: slug, Title: slug, Capacity: capacity, Status: "open", StartsAt: startsAt, EndsAt: endsAt})
	require.NoError(t, err)
	return cohort
}

func countProjectionRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM user_library_projection`).Scan(&count))
	return count
}

func truncateProjection(t *testing.T, db *sql.DB) error {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `TRUNCATE TABLE user_library_projection`)
	return err
}
