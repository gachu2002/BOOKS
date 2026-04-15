package storageengine

import (
	"fmt"
	"sort"
)

type segmentValue struct {
	value   string
	deleted bool
}

type segment struct {
	data map[string]segmentValue
	keys []string
}

type GetStats struct {
	SegmentsChecked int
	Found           bool
	Deleted         bool
}

type Store struct {
	memtable map[string]segmentValue
	segments []segment
}

func NewStore() *Store {
	return &Store{memtable: make(map[string]segmentValue)}
}

func (s *Store) Put(key, value string) {
	s.memtable[key] = segmentValue{value: value}
}

func (s *Store) Delete(key string) {
	s.memtable[key] = segmentValue{deleted: true}
}

func (s *Store) Flush() {
	if len(s.memtable) == 0 {
		return
	}

	keys := make([]string, 0, len(s.memtable))
	data := make(map[string]segmentValue, len(s.memtable))
	for key, value := range s.memtable {
		keys = append(keys, key)
		data[key] = value
	}
	sort.Strings(keys)

	s.segments = append([]segment{{data: data, keys: keys}}, s.segments...)
	s.memtable = make(map[string]segmentValue)
}

func (s *Store) Get(key string) (string, GetStats) {
	if value, ok := s.memtable[key]; ok {
		return getResult(value, GetStats{Found: true, Deleted: value.deleted})
	}

	stats := GetStats{}
	for _, seg := range s.segments {
		stats.SegmentsChecked++
		if !segmentHasKey(seg.keys, key) {
			continue
		}
		value := seg.data[key]
		stats.Found = true
		stats.Deleted = value.deleted
		return getResult(value, stats)
	}

	return "", stats
}

func getResult(value segmentValue, stats GetStats) (string, GetStats) {
	if value.deleted {
		return "", stats
	}
	return value.value, stats
}

func (s *Store) Compact() {
	if len(s.segments) <= 1 {
		return
	}

	merged := make(map[string]segmentValue)
	for _, seg := range s.segments {
		for _, key := range seg.keys {
			if _, exists := merged[key]; exists {
				continue
			}
			merged[key] = seg.data[key]
		}
	}

	keys := make([]string, 0, len(merged))
	compacted := make(map[string]segmentValue)
	for key, value := range merged {
		if value.deleted {
			continue
		}
		keys = append(keys, key)
		compacted[key] = value
	}
	sort.Strings(keys)
	s.segments = []segment{{data: compacted, keys: keys}}
}

func (s *Store) SegmentCount() int {
	return len(s.segments)
}

func segmentHasKey(keys []string, key string) bool {
	idx := sort.SearchStrings(keys, key)
	return idx < len(keys) && keys[idx] == key
}

type OrderRow struct {
	ID         int
	Status     string
	Currency   string
	TotalCents int64
	Day        int
}

type OrderColumns struct {
	IDs        []int
	Statuses   []string
	Currencies []string
	Totals     []int64
	Days       []int
	idIndex    map[int]int
}

func NewOrderColumns(rows []OrderRow) OrderColumns {
	columns := OrderColumns{
		IDs:        make([]int, 0, len(rows)),
		Statuses:   make([]string, 0, len(rows)),
		Currencies: make([]string, 0, len(rows)),
		Totals:     make([]int64, 0, len(rows)),
		Days:       make([]int, 0, len(rows)),
		idIndex:    make(map[int]int, len(rows)),
	}

	for i, row := range rows {
		columns.IDs = append(columns.IDs, row.ID)
		columns.Statuses = append(columns.Statuses, row.Status)
		columns.Currencies = append(columns.Currencies, row.Currency)
		columns.Totals = append(columns.Totals, row.TotalCents)
		columns.Days = append(columns.Days, row.Day)
		columns.idIndex[row.ID] = i
	}

	return columns
}

func RevenueByDayRowStore(rows []OrderRow, day int, currency string) (sum int64, cellsRead int) {
	for _, row := range rows {
		cellsRead += 5
		if row.Status == "paid" && row.Currency == currency && row.Day == day {
			sum += row.TotalCents
		}
	}
	return sum, cellsRead
}

func RevenueByDayColumnStore(columns OrderColumns, day int, currency string) (sum int64, cellsRead int) {
	for i := range columns.IDs {
		cellsRead += 3
		if columns.Statuses[i] == "paid" && columns.Currencies[i] == currency && columns.Days[i] == day {
			sum += columns.Totals[i]
			cellsRead++
		}
	}
	return sum, cellsRead
}

func LookupOrderRowStore(rows []OrderRow, id int) (OrderRow, int, error) {
	for _, row := range rows {
		if row.ID == id {
			return row, 1, nil
		}
	}
	return OrderRow{}, len(rows), fmt.Errorf("row %d not found", id)
}

func LookupOrderColumnStore(columns OrderColumns, id int) (OrderRow, int, error) {
	idx, ok := columns.idIndex[id]
	if !ok {
		return OrderRow{}, 1, fmt.Errorf("row %d not found", id)
	}

	return OrderRow{
		ID:         columns.IDs[idx],
		Status:     columns.Statuses[idx],
		Currency:   columns.Currencies[idx],
		TotalCents: columns.Totals[idx],
		Day:        columns.Days[idx],
	}, 5, nil
}
