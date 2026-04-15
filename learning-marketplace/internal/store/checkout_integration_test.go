package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"learning-marketplace/internal/store"
	"learning-marketplace/internal/testutil"
)

func TestCheckoutCohort_PreventsOversell(t *testing.T) {
	testDB := testutil.NewMigratedPostgres(t, "lm_test")
	s := store.New(testDB.DB)

	user1 := createUserFixture(t, s, "one@example.com")
	user2 := createUserFixture(t, s, "two@example.com")
	product := createLiveProductFixture(t, s, "cohort-product")
	cohort := createCohortFixture(t, s, product.ID, "cohort-one", 1)

	start := make(chan struct{})
	type outcome struct {
		result store.CheckoutResult
		err    error
	}
	outcomes := make(chan outcome, 2)

	go func() {
		<-start
		result, err := s.CheckoutCohort(context.Background(), store.CheckoutParams{
			UserID:          user1.ID,
			CohortID:        cohort.ID,
			PaymentProvider: "demo",
			IdempotencyKey:  "oversell-one",
		})
		outcomes <- outcome{result: result, err: err}
	}()

	go func() {
		<-start
		result, err := s.CheckoutCohort(context.Background(), store.CheckoutParams{
			UserID:          user2.ID,
			CohortID:        cohort.ID,
			PaymentProvider: "demo",
			IdempotencyKey:  "oversell-two",
		})
		outcomes <- outcome{result: result, err: err}
	}()

	close(start)

	first := <-outcomes
	second := <-outcomes

	results := []outcome{first, second}
	successes := 0
	soldOuts := 0
	for _, item := range results {
		switch {
		case item.err == nil:
			successes++
		case item.err == store.ErrSoldOut:
			soldOuts++
		default:
			require.NoError(t, item.err)
		}
	}

	require.Equal(t, 1, successes)
	require.Equal(t, 1, soldOuts)
	require.Equal(t, 1, countPaidSeats(t, testDB.DB, cohort.ID))
	require.Equal(t, 1, countRows(t, testDB.DB, "orders"))
	require.Equal(t, 1, countRows(t, testDB.DB, "payments"))
	require.Equal(t, 1, countRows(t, testDB.DB, "entitlements"))
}

func TestCheckoutCohort_IsIdempotentForSameKey(t *testing.T) {
	testDB := testutil.NewMigratedPostgres(t, "lm_test")
	s := store.New(testDB.DB)

	user := createUserFixture(t, s, "idempotent@example.com")
	product := createLiveProductFixture(t, s, "idempotent-product")
	cohort := createCohortFixture(t, s, product.ID, "idempotent-cohort", 2)

	first, err := s.CheckoutCohort(context.Background(), store.CheckoutParams{
		UserID:            user.ID,
		CohortID:          cohort.ID,
		PaymentProvider:   "demo",
		IdempotencyKey:    "idem-001",
		ProviderPaymentID: stringPtr("pi_demo_001"),
	})
	require.NoError(t, err)
	require.False(t, first.IdempotentReplay)
	require.Equal(t, user.ID, first.Placement.UserRoute.PartitionKey)
	require.Equal(t, cohort.ID, first.Placement.CohortRoute.PartitionKey)
	require.Equal(t, first.Placement.UserRoute.VirtualShard != first.Placement.CohortRoute.VirtualShard, first.Placement.CrossShard)

	second, err := s.CheckoutCohort(context.Background(), store.CheckoutParams{
		UserID:            user.ID,
		CohortID:          cohort.ID,
		PaymentProvider:   "demo",
		IdempotencyKey:    "idem-001",
		ProviderPaymentID: stringPtr("pi_demo_001"),
	})
	require.NoError(t, err)
	require.True(t, second.IdempotentReplay)
	require.Equal(t, first.Order.ID, second.Order.ID)
	require.Equal(t, first.Payment.ID, second.Payment.ID)
	require.Equal(t, first.Entitlement.ID, second.Entitlement.ID)
	require.Equal(t, first.Placement, second.Placement)
	require.Equal(t, 1, countRows(t, testDB.DB, "orders"))
	require.Equal(t, 1, countRows(t, testDB.DB, "payments"))
	require.Equal(t, 1, countRows(t, testDB.DB, "entitlements"))
	require.Equal(t, 1, countPaidSeats(t, testDB.DB, cohort.ID))
}

func TestCheckoutCohort_PromoExhaustionRace(t *testing.T) {
	testDB := testutil.NewMigratedPostgres(t, "lm_test")
	s := store.New(testDB.DB)

	user1 := createUserFixture(t, s, "promo-one@example.com")
	user2 := createUserFixture(t, s, "promo-two@example.com")
	product := createLiveProductFixture(t, s, "promo-product")
	cohort := createCohortFixture(t, s, product.ID, "promo-cohort", 2)
	promo := createPromoFixture(t, s, "ONEUSE", 1)

	start := make(chan struct{})
	type outcome struct {
		result store.CheckoutResult
		err    error
	}
	outcomes := make(chan outcome, 2)

	go func() {
		<-start
		result, err := s.CheckoutCohort(context.Background(), store.CheckoutParams{
			UserID:          user1.ID,
			CohortID:        cohort.ID,
			PromoCode:       stringPtr(promo.Code),
			PaymentProvider: "demo",
			IdempotencyKey:  "promo-race-one",
		})
		outcomes <- outcome{result: result, err: err}
	}()

	go func() {
		<-start
		result, err := s.CheckoutCohort(context.Background(), store.CheckoutParams{
			UserID:          user2.ID,
			CohortID:        cohort.ID,
			PromoCode:       stringPtr(promo.Code),
			PaymentProvider: "demo",
			IdempotencyKey:  "promo-race-two",
		})
		outcomes <- outcome{result: result, err: err}
	}()

	close(start)

	first := <-outcomes
	second := <-outcomes
	results := []outcome{first, second}

	successes := 0
	exhausted := 0
	for _, item := range results {
		switch {
		case item.err == nil:
			successes++
		case item.err == store.ErrPromoCodeExhausted:
			exhausted++
		default:
			require.NoError(t, item.err)
		}
	}

	require.Equal(t, 1, successes)
	require.Equal(t, 1, exhausted)
	require.Equal(t, 1, countRows(t, testDB.DB, "promo_code_redemptions"))
	require.Equal(t, 1, countRows(t, testDB.DB, "payments"))
	require.Equal(t, 1, countRows(t, testDB.DB, "orders"))
}

func createUserFixture(t *testing.T, s *store.Store, email string) store.User {
	t.Helper()

	user, err := s.CreateUser(context.Background(), store.UserCreateParams{
		Email:    email,
		FullName: email,
		Role:     "student",
	})
	require.NoError(t, err)
	return user
}

func createLiveProductFixture(t *testing.T, s *store.Store, slug string) store.Product {
	t.Helper()

	product, err := s.CreateProduct(context.Background(), store.ProductCreateParams{
		Slug:        slug,
		Title:       slug,
		Description: "live cohort product",
		ProductType: "live_cohort",
		PriceCents:  10000,
		Currency:    "USD",
		Published:   true,
	})
	require.NoError(t, err)
	return product
}

func createCohortFixture(t *testing.T, s *store.Store, productID, slug string, capacity int) store.Cohort {
	t.Helper()

	startsAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	endsAt := startsAt.Add(7 * 24 * time.Hour)
	cohort, err := s.CreateCohort(context.Background(), store.CohortCreateParams{
		ProductID: productID,
		Slug:      slug,
		Title:     slug,
		Capacity:  capacity,
		Status:    "open",
		StartsAt:  startsAt,
		EndsAt:    endsAt,
	})
	require.NoError(t, err)
	return cohort
}

func createPromoFixture(t *testing.T, s *store.Store, code string, maxRedemptions int) store.PromoCode {
	t.Helper()

	promo, err := s.CreatePromoCode(context.Background(), store.PromoCodeCreateParams{
		Code:           code,
		DiscountType:   "percent",
		DiscountValue:  10,
		MaxRedemptions: intPtr(maxRedemptions),
	})
	require.NoError(t, err)
	return promo
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()

	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count))
	return count
}

func countPaidSeats(t *testing.T, db *sql.DB, cohortID string) int {
	t.Helper()

	const query = `
SELECT COALESCE(SUM(oi.quantity), 0)
FROM order_items oi
JOIN orders o ON o.id = oi.order_id
WHERE oi.cohort_id = $1
  AND o.status = 'paid'`

	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), query, cohortID).Scan(&count))
	return count
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}
