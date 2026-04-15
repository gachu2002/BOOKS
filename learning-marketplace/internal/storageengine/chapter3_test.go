package storageengine_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"learning-marketplace/internal/storageengine"
)

func TestStore_CompactionKeepsNewestValueAndReducesReadAmplification(t *testing.T) {
	store := storageengine.NewStore()

	store.Put("promo:launch", "10")
	store.Flush()
	store.Put("promo:launch", "15")
	store.Flush()
	store.Put("promo:vip", "25")
	store.Flush()

	value, before := store.Get("promo:launch")
	require.Equal(t, "15", value)
	require.Equal(t, 2, before.SegmentsChecked)
	require.Equal(t, 3, store.SegmentCount())

	store.Compact()

	value, after := store.Get("promo:launch")
	require.Equal(t, "15", value)
	require.Equal(t, 1, after.SegmentsChecked)
	require.Equal(t, 1, store.SegmentCount())
}

func TestStore_CompactionDropsDeletedKeys(t *testing.T) {
	store := storageengine.NewStore()

	store.Put("product:draft", "visible")
	store.Flush()
	store.Delete("product:draft")
	store.Flush()
	store.Compact()

	value, stats := store.Get("product:draft")
	require.Empty(t, value)
	require.False(t, stats.Found)
	require.False(t, stats.Deleted)
}

func TestColumnStore_ShowsAnalyticalBenefitAndPointLookupTradeOff(t *testing.T) {
	rows := []storageengine.OrderRow{
		{ID: 1, Status: "paid", Currency: "USD", TotalCents: 1000, Day: 1},
		{ID: 2, Status: "pending", Currency: "USD", TotalCents: 2000, Day: 1},
		{ID: 3, Status: "paid", Currency: "USD", TotalCents: 3000, Day: 1},
		{ID: 4, Status: "paid", Currency: "EUR", TotalCents: 4000, Day: 1},
	}
	columns := storageengine.NewOrderColumns(rows)

	rowSum, rowCells := storageengine.RevenueByDayRowStore(rows, 1, "USD")
	columnSum, columnCells := storageengine.RevenueByDayColumnStore(columns, 1, "USD")
	require.EqualValues(t, 4000, rowSum)
	require.Equal(t, rowSum, columnSum)
	require.Greater(t, rowCells, columnCells)

	_, rowLookupCells, err := storageengine.LookupOrderRowStore(rows, 3)
	require.NoError(t, err)
	_, columnLookupCells, err := storageengine.LookupOrderColumnStore(columns, 3)
	require.NoError(t, err)
	require.Less(t, rowLookupCells, columnLookupCells)
}
