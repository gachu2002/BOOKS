package main

import (
	"fmt"

	"learning-marketplace/internal/storageengine"
)

func main() {
	appendOnlyDemo()
	columnStoreDemo()
}

func appendOnlyDemo() {
	store := storageengine.NewStore()
	store.Put("promo:launch", "10")
	store.Flush()
	store.Put("promo:launch", "15")
	store.Flush()
	store.Put("promo:vip", "25")
	store.Flush()

	value, before := store.Get("promo:launch")
	fmt.Println("== Append-only segments and compaction ==")
	fmt.Printf("before compaction: value=%s segments=%d segments_checked=%d\n", value, store.SegmentCount(), before.SegmentsChecked)

	store.Compact()
	value, after := store.Get("promo:launch")
	fmt.Printf("after compaction:  value=%s segments=%d segments_checked=%d\n", value, store.SegmentCount(), after.SegmentsChecked)
	fmt.Println("trade-off: appends make writes simple, but reads may touch multiple immutable segments until compaction rewrites them")
	fmt.Println()
}

func columnStoreDemo() {
	rows := []storageengine.OrderRow{
		{ID: 1, Status: "paid", Currency: "USD", TotalCents: 1000, Day: 1},
		{ID: 2, Status: "pending", Currency: "USD", TotalCents: 2000, Day: 1},
		{ID: 3, Status: "paid", Currency: "USD", TotalCents: 3000, Day: 1},
		{ID: 4, Status: "paid", Currency: "EUR", TotalCents: 4000, Day: 1},
		{ID: 5, Status: "paid", Currency: "USD", TotalCents: 5000, Day: 2},
	}
	columns := storageengine.NewOrderColumns(rows)

	rowSum, rowCells := storageengine.RevenueByDayRowStore(rows, 1, "USD")
	columnSum, columnCells := storageengine.RevenueByDayColumnStore(columns, 1, "USD")
	_, rowLookupCells, _ := storageengine.LookupOrderRowStore(rows, 3)
	_, columnLookupCells, _ := storageengine.LookupOrderColumnStore(columns, 3)

	fmt.Println("== Row-oriented vs column-oriented reads ==")
	fmt.Printf("daily revenue query: row_store_sum=%d row_cells_read=%d column_store_sum=%d column_cells_read=%d\n", rowSum, rowCells, columnSum, columnCells)
	fmt.Printf("point lookup query: row_cells_read=%d column_cells_read=%d\n", rowLookupCells, columnLookupCells)
	fmt.Println("trade-off: columnar layout helps narrow analytical scans, while row layout is simpler for point reads that need the whole record")
}
