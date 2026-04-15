package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"learning-marketplace/internal/store"
)

func TestRouteUser_UsesStableVirtualShard(t *testing.T) {
	s := store.New(nil)

	first := s.RouteUser("user-123")
	second := s.RouteUser("user-123")
	other := s.RouteUser("user-456")

	require.Equal(t, "hash(user_id)", first.Strategy)
	require.Equal(t, "user-123", first.PartitionKey)
	require.Equal(t, 16, first.VirtualShardCount)
	require.Equal(t, first, second)
	require.NotEqual(t, first.PartitionKey, other.PartitionKey)
}

func TestCheckoutPlacement_ExposesUserAndCohortRouting(t *testing.T) {
	s := store.New(nil)
	placement := s.CheckoutPlacement(store.CheckoutParams{UserID: "user-123", CohortID: "cohort-123"})

	require.Equal(t, s.RouteUser("user-123"), placement.UserRoute)
	require.Equal(t, s.RouteCohort("cohort-123"), placement.CohortRoute)
	require.Equal(t, placement.UserRoute.VirtualShard != placement.CohortRoute.VirtualShard, placement.CrossShard)
}
