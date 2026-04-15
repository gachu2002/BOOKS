package partitioning

import (
	"fmt"
	"hash/fnv"
	"sort"
)

type RangePartition struct {
	Name string
	Min  int
	Max  int
}

type RangeRouter struct {
	Partitions []RangePartition
}

func (r RangeRouter) Route(key int) (string, error) {
	for _, partition := range r.Partitions {
		if key >= partition.Min && key < partition.Max {
			return partition.Name, nil
		}
	}

	return "", fmt.Errorf("no range partition for key %d", key)
}

func (r RangeRouter) RouteRange(minKey, maxKey int) []string {
	shards := make([]string, 0, len(r.Partitions))
	for _, partition := range r.Partitions {
		if maxKey <= partition.Min || minKey >= partition.Max {
			continue
		}
		shards = append(shards, partition.Name)
	}

	return shards
}

type HashRouter struct {
	Shards []string
}

func (r HashRouter) Route(key string) string {
	if len(r.Shards) == 0 {
		return ""
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return r.Shards[uint(h.Sum32())%uint(len(r.Shards))]
}

type Load struct {
	PerShard map[string]int
}

func CountIntKeys(router RangeRouter, keys []int) (Load, error) {
	load := Load{PerShard: make(map[string]int, len(router.Partitions))}
	for _, partition := range router.Partitions {
		load.PerShard[partition.Name] = 0
	}

	for _, key := range keys {
		shard, err := router.Route(key)
		if err != nil {
			return Load{}, err
		}
		load.PerShard[shard]++
	}

	return load, nil
}

func CountStringKeys(router HashRouter, keys []string) Load {
	load := Load{PerShard: make(map[string]int, len(router.Shards))}
	for _, shard := range router.Shards {
		load.PerShard[shard] = 0
	}

	for _, key := range keys {
		load.PerShard[router.Route(key)]++
	}

	return load
}

func (l Load) HottestShard() (string, int) {
	var hottest string
	var max int
	for shard, count := range l.PerShard {
		if hottest == "" || count > max {
			hottest = shard
			max = count
		}
	}

	return hottest, max
}

func (l Load) SortedShards() []string {
	shards := make([]string, 0, len(l.PerShard))
	for shard := range l.PerShard {
		shards = append(shards, shard)
	}
	sort.Strings(shards)
	return shards
}

type VirtualShardAssignment struct {
	NodeByVirtualShard []string
	Nodes              []string
}

func NewVirtualShardAssignment(virtualShardCount int, nodes []string) VirtualShardAssignment {
	assignment := VirtualShardAssignment{
		NodeByVirtualShard: make([]string, virtualShardCount),
		Nodes:              append([]string(nil), nodes...),
	}

	for shard := 0; shard < virtualShardCount; shard++ {
		assignment.NodeByVirtualShard[shard] = nodes[shard%len(nodes)]
	}

	return assignment
}

func (a VirtualShardAssignment) VirtualShardForKey(key string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(uint(h.Sum32()) % uint(len(a.NodeByVirtualShard)))
}

func (a VirtualShardAssignment) NodeForKey(key string) string {
	return a.NodeByVirtualShard[a.VirtualShardForKey(key)]
}

func (a VirtualShardAssignment) Rebalance(nodes []string) VirtualShardAssignment {
	next := VirtualShardAssignment{
		NodeByVirtualShard: make([]string, len(a.NodeByVirtualShard)),
		Nodes:              append([]string(nil), nodes...),
	}

	for shard := range next.NodeByVirtualShard {
		next.NodeByVirtualShard[shard] = nodes[shard%len(nodes)]
	}

	return next
}

func (a VirtualShardAssignment) MovedVirtualShards(next VirtualShardAssignment) []int {
	moved := make([]int, 0)
	for shard := range a.NodeByVirtualShard {
		if a.NodeByVirtualShard[shard] != next.NodeByVirtualShard[shard] {
			moved = append(moved, shard)
		}
	}
	return moved
}

type CohortRecord struct {
	ID       string
	Status   string
	CohortID string
}

type SearchPlan struct {
	ShardsTouched []string
	RecordIDs     []string
}

func LocalSecondaryIndexSearch(primary map[string][]CohortRecord, status string) SearchPlan {
	shardsTouched := make([]string, 0, len(primary))
	ids := make([]string, 0)
	for shard, records := range primary {
		shardsTouched = append(shardsTouched, shard)
		for _, record := range records {
			if record.Status == status {
				ids = append(ids, record.ID)
			}
		}
	}
	sort.Strings(shardsTouched)
	sort.Strings(ids)
	return SearchPlan{ShardsTouched: shardsTouched, RecordIDs: ids}
}

type GlobalSecondaryIndex struct {
	Router      HashRouter
	IDsByStatus map[string][]string
}

func BuildGlobalSecondaryIndex(router HashRouter, primary map[string][]CohortRecord) GlobalSecondaryIndex {
	idsByStatus := make(map[string][]string)
	for _, records := range primary {
		for _, record := range records {
			idsByStatus[record.Status] = append(idsByStatus[record.Status], record.ID)
		}
	}

	for status := range idsByStatus {
		sort.Strings(idsByStatus[status])
	}

	return GlobalSecondaryIndex{Router: router, IDsByStatus: idsByStatus}
}

func (g GlobalSecondaryIndex) Search(status string) SearchPlan {
	ids := append([]string(nil), g.IDsByStatus[status]...)
	shard := g.Router.Route(status)
	return SearchPlan{ShardsTouched: []string{shard}, RecordIDs: ids}
}

func SecondaryIndexWriteTargets(record CohortRecord, primaryRouter, indexRouter HashRouter) []string {
	targets := []string{primaryRouter.Route(record.CohortID), indexRouter.Route(record.Status)}
	sort.Strings(targets)
	if len(targets) == 2 && targets[0] == targets[1] {
		return targets[:1]
	}
	return targets
}
