package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"learning-marketplace/internal/projector"
	"learning-marketplace/internal/store"
	"learning-marketplace/internal/testutil"
)

func TestUserShards_RoutesUserReadsToOwningStore(t *testing.T) {
	shardA := testutil.NewMigratedPostgres(t, "lm_user_shard_a")
	shardB := testutil.NewMigratedPostgres(t, "lm_user_shard_b")

	storeA := store.New(shardA.DB)
	storeB := store.New(shardB.DB)
	shards := store.NewUserShards(16, []store.NamedStore{{Name: "shard-a", Store: storeA}, {Name: "shard-b", Store: storeB}})

	ctx := context.Background()
	userA := createUserForOwner(t, storeA, shards, "shard-a", "a")
	userB := createUserForOwner(t, storeB, shards, "shard-b", "b")

	resultA := checkoutOnStore(t, storeA, userA.ID, "alpha")
	resultB := checkoutOnStore(t, storeB, userB.ID, "bravo")

	itemsA, selectionA, err := shards.ListEntitlementsByUser(ctx, userA.ID, store.Pagination{Limit: 10, Offset: 0})
	require.NoError(t, err)
	require.Equal(t, "shard-a", selectionA.Owner)
	require.Len(t, itemsA, 1)
	require.Equal(t, resultA.Order.ID, itemsA[0].Entitlement.OrderID)
	require.Equal(t, resultA.Product.ID, itemsA[0].Entitlement.ProductID)

	itemsB, selectionB, err := shards.ListEntitlementsByUser(ctx, userB.ID, store.Pagination{Limit: 10, Offset: 0})
	require.NoError(t, err)
	require.Equal(t, "shard-b", selectionB.Owner)
	require.Len(t, itemsB, 1)
	require.Equal(t, resultB.Order.ID, itemsB[0].Entitlement.OrderID)
	require.Equal(t, resultB.Product.ID, itemsB[0].Entitlement.ProductID)
}

func TestUserShards_RoutesProjectionReadsToOwningStore(t *testing.T) {
	shardA := testutil.NewMigratedPostgres(t, "lm_library_shard_a")
	shardB := testutil.NewMigratedPostgres(t, "lm_library_shard_b")

	storeA := store.New(shardA.DB)
	storeB := store.New(shardB.DB)
	shards := store.NewUserShards(16, []store.NamedStore{{Name: "shard-a", Store: storeA}, {Name: "shard-b", Store: storeB}})

	ctx := context.Background()
	userA := createUserForOwner(t, storeA, shards, "shard-a", "projection-a")
	userB := createUserForOwner(t, storeB, shards, "shard-b", "projection-b")

	resultA := checkoutOnStore(t, storeA, userA.ID, "library-alpha")
	resultB := checkoutOnStore(t, storeB, userB.ID, "library-bravo")

	_, err := projector.NewUserLibraryProjector(shardA.DB).ProcessPending(ctx, 10)
	require.NoError(t, err)
	_, err = projector.NewUserLibraryProjector(shardB.DB).ProcessPending(ctx, 10)
	require.NoError(t, err)

	itemsA, selectionA, err := shards.ListUserLibrary(ctx, userA.ID, store.Pagination{Limit: 10, Offset: 0})
	require.NoError(t, err)
	require.Equal(t, "shard-a", selectionA.Owner)
	require.Len(t, itemsA, 1)
	require.Equal(t, resultA.Order.ID, itemsA[0].OrderID)
	require.Equal(t, resultA.Product.Slug, itemsA[0].ProductSlug)

	itemsB, selectionB, err := shards.ListUserLibrary(ctx, userB.ID, store.Pagination{Limit: 10, Offset: 0})
	require.NoError(t, err)
	require.Equal(t, "shard-b", selectionB.Owner)
	require.Len(t, itemsB, 1)
	require.Equal(t, resultB.Order.ID, itemsB[0].OrderID)
	require.Equal(t, resultB.Product.Slug, itemsB[0].ProductSlug)
}

func createUserForOwner(t *testing.T, s *store.Store, shards *store.UserShards, owner string, prefix string) store.User {
	t.Helper()

	for attempt := 0; attempt < 64; attempt++ {
		user := createUserFixture(t, s, fmt.Sprintf("%s-%02d@example.com", prefix, attempt))
		_, selection := shards.Resolve(user.ID)
		if selection.Owner == owner {
			return user
		}
	}

	t.Fatalf("failed to create user routed to %s", owner)
	return store.User{}
}

func checkoutOnStore(t *testing.T, s *store.Store, userID string, slug string) store.CheckoutResult {
	t.Helper()

	product := createLiveProductFixture(t, s, slug+"-product")
	cohort := createCohortFixture(t, s, product.ID, slug+"-cohort", 5)
	result, err := s.CheckoutCohort(context.Background(), store.CheckoutParams{
		UserID:          userID,
		CohortID:        cohort.ID,
		PaymentProvider: "demo",
		IdempotencyKey:  slug + "-checkout",
	})
	require.NoError(t, err)
	return result
}
