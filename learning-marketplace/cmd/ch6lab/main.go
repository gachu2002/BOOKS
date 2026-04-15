package main

import (
	"fmt"

	"learning-marketplace/internal/partitioning"
)

func main() {
	hotspotDemo()
	rebalanceDemo()
	secondaryIndexDemo()
}

func hotspotDemo() {
	rangeRouter := partitioning.RangeRouter{Partitions: []partitioning.RangePartition{
		{Name: "checkouts-0-100", Min: 0, Max: 100},
		{Name: "checkouts-100-200", Min: 100, Max: 200},
		{Name: "checkouts-200-300", Min: 200, Max: 300},
	}}

	newestWrites := make([]int, 0, 20)
	for checkoutID := 280; checkoutID < 300; checkoutID++ {
		newestWrites = append(newestWrites, checkoutID)
	}

	rangeLoad, err := partitioning.CountIntKeys(rangeRouter, newestWrites)
	if err != nil {
		panic(err)
	}

	hashKeys := make([]string, 0, 20)
	for _, checkoutID := range newestWrites {
		hashKeys = append(hashKeys, fmt.Sprintf("checkout-%03d", checkoutID))
	}
	hashRouter := partitioning.HashRouter{Shards: []string{"shard-a", "shard-b", "shard-c", "shard-d"}}
	hashLoad := partitioning.CountStringKeys(hashRouter, hashKeys)

	fmt.Println("== Key range vs hash sharding ==")
	fmt.Println("newest checkout IDs all land in the same key-range partition, which models DDIA's 'latest timestamp' hot spot")
	printLoad("range", rangeLoad)
	printLoad("hash ", hashLoad)
	fmt.Println("trade-off: range partitioning keeps nearby keys together for scans, while hashing spreads bursty writes but destroys range locality")
	fmt.Println()
}

func rebalanceDemo() {
	before := partitioning.NewVirtualShardAssignment(12, []string{"node-a", "node-b"})
	after := before.Rebalance([]string{"node-a", "node-b", "node-c"})
	key := "cohort:hot-seat"

	fmt.Println("== Rebalancing with fixed virtual shards ==")
	fmt.Printf("key=%s virtual_shard=%d before_node=%s after_node=%s moved_virtual_shards=%v\n",
		key,
		before.VirtualShardForKey(key),
		before.NodeForKey(key),
		after.NodeForKey(key),
		before.MovedVirtualShards(after),
	)
	fmt.Println("trade-off: keys stay mapped to the same virtual shard, but only shard ownership moves when a new node is added")
	fmt.Println()
}

func secondaryIndexDemo() {
	primary := map[string][]partitioning.CohortRecord{
		"primary-a": {{ID: "cohort-1", Status: "open", CohortID: "cohort-1"}},
		"primary-b": {{ID: "cohort-2", Status: "closed", CohortID: "cohort-2"}, {ID: "cohort-3", Status: "open", CohortID: "cohort-3"}},
		"primary-c": {{ID: "cohort-4", Status: "draft", CohortID: "cohort-4"}},
	}

	local := partitioning.LocalSecondaryIndexSearch(primary, "open")
	global := partitioning.BuildGlobalSecondaryIndex(partitioning.HashRouter{Shards: []string{"index-a", "index-b"}}, primary).Search("open")
	writeTargets := partitioning.SecondaryIndexWriteTargets(
		partitioning.CohortRecord{ID: "cohort-5", Status: "open", CohortID: "cohort-5"},
		partitioning.HashRouter{Shards: []string{"primary-a", "primary-b", "primary-c"}},
		partitioning.HashRouter{Shards: []string{"index-a", "index-b"}},
	)

	fmt.Println("== Local vs global secondary indexes ==")
	fmt.Printf("local-index search status=open touches=%v ids=%v\n", local.ShardsTouched, local.RecordIDs)
	fmt.Printf("global-index search status=open touches=%v ids=%v\n", global.ShardsTouched, global.RecordIDs)
	fmt.Printf("one write to cohort-5 touches primary+index shards=%v\n", writeTargets)
	fmt.Println("trade-off: local indexes keep writes cheap but fan out reads; global indexes narrow the search hop but complicate writes and still need record fetches")
}

func printLoad(label string, load partitioning.Load) {
	fmt.Printf("%s load:", label)
	for _, shard := range load.SortedShards() {
		fmt.Printf(" %s=%d", shard, load.PerShard[shard])
	}
	hottest, count := load.HottestShard()
	fmt.Printf(" hottest=%s(%d)\n", hottest, count)
}
