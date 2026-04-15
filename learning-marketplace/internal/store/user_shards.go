package store

import "context"

type NamedStore struct {
	Name  string
	Store *Store
}

type UserShardSelection struct {
	Route ShardRoute `json:"route"`
	Owner string     `json:"owner"`
}

type UserShards struct {
	planner ShardPlanner
	nodes   []NamedStore
}

func NewUserShards(virtualShards int, nodes []NamedStore) *UserShards {
	cleaned := make([]NamedStore, 0, len(nodes))
	for _, node := range nodes {
		if node.Store == nil {
			continue
		}
		if node.Name == "" {
			node.Name = "primary"
		}
		cleaned = append(cleaned, node)
	}
	if len(cleaned) == 0 {
		return nil
	}

	return &UserShards{planner: NewShardPlanner(virtualShards), nodes: cleaned}
}

func (s *UserShards) Resolve(userID string) (*Store, UserShardSelection) {
	if s == nil || len(s.nodes) == 0 {
		return nil, UserShardSelection{}
	}

	route := s.planner.route("hash(user_id)", userID)
	node := s.nodes[route.VirtualShard%len(s.nodes)]

	return node.Store, UserShardSelection{Route: route, Owner: node.Name}
}

func (s *UserShards) ListEntitlementsByUser(ctx context.Context, userID string, page Pagination) ([]EntitlementWithProduct, UserShardSelection, error) {
	selectedStore, selection := s.Resolve(userID)
	items, err := selectedStore.ListEntitlementsByUser(ctx, userID, page)
	return items, selection, err
}

func (s *UserShards) ListUserLibrary(ctx context.Context, userID string, page Pagination) ([]UserLibraryItem, UserShardSelection, error) {
	selectedStore, selection := s.Resolve(userID)
	items, err := selectedStore.ListUserLibrary(ctx, userID, page)
	return items, selection, err
}
