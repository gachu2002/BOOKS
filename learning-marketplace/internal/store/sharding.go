package store

import "hash/fnv"

type ShardRoute struct {
	Strategy          string `json:"strategy"`
	PartitionKey      string `json:"partition_key"`
	VirtualShard      int    `json:"virtual_shard"`
	VirtualShardCount int    `json:"virtual_shard_count"`
}

type CheckoutPlacement struct {
	UserRoute   ShardRoute `json:"user_route"`
	CohortRoute ShardRoute `json:"cohort_route"`
	CrossShard  bool       `json:"cross_shard"`
}

type ShardPlanner struct {
	virtualShards int
}

func NewShardPlanner(virtualShards int) ShardPlanner {
	if virtualShards < 1 {
		virtualShards = 1
	}

	return ShardPlanner{virtualShards: virtualShards}
}

func (p ShardPlanner) route(strategy, partitionKey string) ShardRoute {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(partitionKey))

	return ShardRoute{
		Strategy:          strategy,
		PartitionKey:      partitionKey,
		VirtualShard:      int(hasher.Sum32() % uint32(p.virtualShards)),
		VirtualShardCount: p.virtualShards,
	}
}

func (s *Store) RouteUser(userID string) ShardRoute {
	return s.shardPlanner.route("hash(user_id)", userID)
}

func (s *Store) RouteCohort(cohortID string) ShardRoute {
	return s.shardPlanner.route("hash(cohort_id)", cohortID)
}

// CheckoutPlacement makes the likely partition keys explicit on the real checkout flow.
func (s *Store) CheckoutPlacement(params CheckoutParams) CheckoutPlacement {
	userRoute := s.RouteUser(params.UserID)
	cohortRoute := s.RouteCohort(params.CohortID)

	return CheckoutPlacement{
		UserRoute:   userRoute,
		CohortRoute: cohortRoute,
		CrossShard:  userRoute.VirtualShard != cohortRoute.VirtualShard,
	}
}
