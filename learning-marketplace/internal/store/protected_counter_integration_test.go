package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"learning-marketplace/internal/store"
	"learning-marketplace/internal/testutil"
)

func TestProtectedCounter_RejectsStaleFencingTokens(t *testing.T) {
	testDB := testutil.NewMigratedPostgres(t, "lm_test")
	s := store.New(testDB.DB)

	first, err := s.ApplyFencedIncrement(context.Background(), "email-sender", "worker-a", 100)
	require.NoError(t, err)
	require.EqualValues(t, 1, first.Value)
	require.EqualValues(t, 100, first.LastFencingToken)

	second, err := s.ApplyFencedIncrement(context.Background(), "email-sender", "worker-b", 101)
	require.NoError(t, err)
	require.EqualValues(t, 2, second.Value)
	require.EqualValues(t, 101, second.LastFencingToken)

	_, err = s.ApplyFencedIncrement(context.Background(), "email-sender", "worker-a", 100)
	require.ErrorIs(t, err, store.ErrStaleFencingToken)

	current, err := s.GetProtectedCounter(context.Background(), "email-sender")
	require.NoError(t, err)
	require.EqualValues(t, 2, current.Value)
	require.EqualValues(t, 101, current.LastFencingToken)
	require.NotNil(t, current.LastHolder)
	require.Equal(t, "worker-b", *current.LastHolder)
}
