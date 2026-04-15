package analytics_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"learning-marketplace/internal/analytics"
	"learning-marketplace/internal/store"
	"learning-marketplace/internal/testutil"
)

func TestReporter_RebuildsDailyRevenueAndCohortFill(t *testing.T) {
	testDB := testutil.NewMigratedPostgres(t, "lm_reports_test")
	s := store.New(testDB.DB)
	reporter := analytics.NewReporter(testDB.DB)

	user1 := createReportsUserFixture(t, s, "report-one@example.com")
	user2 := createReportsUserFixture(t, s, "report-two@example.com")
	product := createReportsProductFixture(t, s, "report-product")
	cohort := createReportsCohortFixture(t, s, product.ID, "report-cohort", 4)
	promo := createReportsPromoFixture(t, s, "REPORT10", 10)

	_, err := s.CheckoutCohort(context.Background(), store.CheckoutParams{UserID: user1.ID, CohortID: cohort.ID, PaymentProvider: "demo", IdempotencyKey: "report-001"})
	require.NoError(t, err)

	_, err = s.CheckoutCohort(context.Background(), store.CheckoutParams{UserID: user2.ID, CohortID: cohort.ID, PromoCode: stringPtr(promo.Code), PaymentProvider: "demo", IdempotencyKey: "report-002"})
	require.NoError(t, err)

	require.NoError(t, reporter.Rebuild(context.Background(), analytics.RebuildMetadata{RebuiltBy: "test-batch", FencingToken: 41}))

	daily, err := reporter.ListDailyRevenue(context.Background(), 10, 0)
	require.NoError(t, err)
	require.Len(t, daily, 1)
	require.Equal(t, 2, daily[0].OrdersCount)
	require.EqualValues(t, 20000, daily[0].GrossRevenueCents)
	require.EqualValues(t, 1000, daily[0].DiscountCents)
	require.EqualValues(t, 19000, daily[0].NetRevenueCents)
	require.Equal(t, "test-batch", daily[0].RebuiltBy)
	require.EqualValues(t, 41, daily[0].FencingToken)

	cohortFill, err := reporter.ListCohortFill(context.Background(), 10, 0)
	require.NoError(t, err)
	require.Len(t, cohortFill, 1)
	require.Equal(t, cohort.ID, cohortFill[0].CohortID)
	require.Equal(t, 4, cohortFill[0].Capacity)
	require.Equal(t, 2, cohortFill[0].SoldSeats)
	require.EqualValues(t, 50.0, cohortFill[0].FillRatePercent)
	require.EqualValues(t, 20000, cohortFill[0].RevenueCents)
	require.Equal(t, "test-batch", cohortFill[0].RebuiltBy)
	require.EqualValues(t, 41, cohortFill[0].FencingToken)
}

func createReportsUserFixture(t *testing.T, s *store.Store, email string) store.User {
	t.Helper()
	user, err := s.CreateUser(context.Background(), store.UserCreateParams{Email: email, FullName: email, Role: "student"})
	require.NoError(t, err)
	return user
}

func createReportsProductFixture(t *testing.T, s *store.Store, slug string) store.Product {
	t.Helper()
	product, err := s.CreateProduct(context.Background(), store.ProductCreateParams{Slug: slug, Title: slug, Description: "analytics", ProductType: "live_cohort", PriceCents: 10000, Currency: "USD", Published: true})
	require.NoError(t, err)
	return product
}

func createReportsCohortFixture(t *testing.T, s *store.Store, productID, slug string, capacity int) store.Cohort {
	t.Helper()
	startsAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	endsAt := startsAt.Add(7 * 24 * time.Hour)
	cohort, err := s.CreateCohort(context.Background(), store.CohortCreateParams{ProductID: productID, Slug: slug, Title: slug, Capacity: capacity, Status: "open", StartsAt: startsAt, EndsAt: endsAt})
	require.NoError(t, err)
	return cohort
}

func createReportsPromoFixture(t *testing.T, s *store.Store, code string, maxRedemptions int) store.PromoCode {
	t.Helper()
	promo, err := s.CreatePromoCode(context.Background(), store.PromoCodeCreateParams{Code: code, DiscountType: "percent", DiscountValue: 10, MaxRedemptions: intPtr(maxRedemptions)})
	require.NoError(t, err)
	return promo
}

func stringPtr(value string) *string { return &value }
func intPtr(value int) *int          { return &value }
