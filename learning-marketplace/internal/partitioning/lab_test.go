package partitioning_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"learning-marketplace/internal/partitioning"
)

func TestRangeShardingBySequentialKeysCreatesHotNewestShard(t *testing.T) {
	router := partitioning.RangeRouter{Partitions: []partitioning.RangePartition{
		{Name: "checkouts-0-100", Min: 0, Max: 100},
		{Name: "checkouts-100-200", Min: 100, Max: 200},
		{Name: "checkouts-200-300", Min: 200, Max: 300},
	}}

	keys := make([]int, 0, 20)
	for checkoutID := 280; checkoutID < 300; checkoutID++ {
		keys = append(keys, checkoutID)
	}

	load, err := partitioning.CountIntKeys(router, keys)
	require.NoError(t, err)

	require.Equal(t, 0, load.PerShard["checkouts-0-100"])
	require.Equal(t, 0, load.PerShard["checkouts-100-200"])
	require.Equal(t, 20, load.PerShard["checkouts-200-300"])

	touched := router.RouteRange(280, 300)
	require.Equal(t, []string{"checkouts-200-300"}, touched)
}

func TestHashShardingSpreadsSequentialKeysAcrossShards(t *testing.T) {
	router := partitioning.HashRouter{Shards: []string{"shard-a", "shard-b", "shard-c", "shard-d"}}

	keys := make([]string, 0, 100)
	for checkoutID := 0; checkoutID < 100; checkoutID++ {
		keys = append(keys, fmt.Sprintf("checkout-%03d", checkoutID))
	}

	load := partitioning.CountStringKeys(router, keys)

	counts := make([]int, 0, len(router.Shards))
	for _, shard := range router.Shards {
		counts = append(counts, load.PerShard[shard])
	}

	require.ElementsMatch(t, []int{25, 25, 25, 25}, counts)
	for _, count := range counts {
		require.Greater(t, count, 0)
	}
}

func TestVirtualShardsRebalanceByMovingShardOwnershipOnly(t *testing.T) {
	before := partitioning.NewVirtualShardAssignment(12, []string{"node-a", "node-b"})
	after := before.Rebalance([]string{"node-a", "node-b", "node-c"})

	key := "cohort:hot-seat"
	require.Equal(t, before.VirtualShardForKey(key), after.VirtualShardForKey(key), "key should stay in the same virtual shard")
	require.NotEqual(t, before.NodeForKey(key), after.NodeForKey(key), "rebalance may move the virtual shard to a different node")

	moved := before.MovedVirtualShards(after)
	require.Len(t, moved, 8)
	require.NotEqual(t, before.NodeByVirtualShard, after.NodeByVirtualShard)
}

func TestSecondaryIndexTradeoffsStayVisible(t *testing.T) {
	primary := map[string][]partitioning.CohortRecord{
		"primary-a": {{ID: "cohort-1", Status: "open", CohortID: "cohort-1"}},
		"primary-b": {{ID: "cohort-2", Status: "closed", CohortID: "cohort-2"}, {ID: "cohort-3", Status: "open", CohortID: "cohort-3"}},
		"primary-c": {{ID: "cohort-4", Status: "draft", CohortID: "cohort-4"}},
	}

	local := partitioning.LocalSecondaryIndexSearch(primary, "open")
	require.Equal(t, []string{"primary-a", "primary-b", "primary-c"}, local.ShardsTouched)
	require.Equal(t, []string{"cohort-1", "cohort-3"}, local.RecordIDs)

	global := partitioning.BuildGlobalSecondaryIndex(partitioning.HashRouter{Shards: []string{"index-a", "index-b"}}, primary)
	globalPlan := global.Search("open")
	require.Len(t, globalPlan.ShardsTouched, 1)
	require.Equal(t, []string{"cohort-1", "cohort-3"}, globalPlan.RecordIDs)

	targets := partitioning.SecondaryIndexWriteTargets(
		partitioning.CohortRecord{ID: "cohort-5", Status: "open", CohortID: "cohort-5"},
		partitioning.HashRouter{Shards: []string{"primary-a", "primary-b", "primary-c"}},
		partitioning.HashRouter{Shards: []string{"index-a", "index-b"}},
	)
	require.GreaterOrEqual(t, len(targets), 1)
	require.LessOrEqual(t, len(targets), 2)
	// A global secondary index narrows the read fan-out, but a write may need to touch both the primary shard and an index shard.
	require.NotEmpty(t, targets)
	if len(targets) == 2 {
		require.NotEqual(t, targets[0], targets[1])
	}
}
