package store

import "database/sql"

type Store struct {
	db           *sql.DB
	shardPlanner ShardPlanner
}

func New(db *sql.DB) *Store {
	return &Store{db: db, shardPlanner: NewShardPlanner(16)}
}

func (s *Store) DB() *sql.DB {
	return s.db
}
