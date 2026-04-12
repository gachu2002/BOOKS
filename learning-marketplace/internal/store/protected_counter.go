package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrStaleFencingToken = errors.New("stale fencing token")

// ProtectedCounter simulates downstream state that must reject stale leaseholders.
type ProtectedCounter struct {
	ResourceName     string    `json:"resource_name"`
	Value            int64     `json:"value"`
	LastFencingToken int64     `json:"last_fencing_token"`
	LastHolder       *string   `json:"last_holder,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ApplyFencedIncrement accepts only strictly newer fencing tokens for the same resource.
func (s *Store) ApplyFencedIncrement(ctx context.Context, resourceName string, holder string, fencingToken int64) (ProtectedCounter, error) {
	const query = `
INSERT INTO protected_counters (resource_name, value, last_fencing_token, last_holder)
VALUES ($1, 1, $2, $3)
ON CONFLICT (resource_name) DO UPDATE
SET value = protected_counters.value + 1,
    last_fencing_token = EXCLUDED.last_fencing_token,
    last_holder = EXCLUDED.last_holder,
    updated_at = now()
WHERE EXCLUDED.last_fencing_token > protected_counters.last_fencing_token
RETURNING resource_name, value, last_fencing_token, last_holder, updated_at`

	var counter ProtectedCounter
	err := s.db.QueryRowContext(ctx, query, resourceName, fencingToken, holder).Scan(
		&counter.ResourceName,
		&counter.Value,
		&counter.LastFencingToken,
		&counter.LastHolder,
		&counter.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProtectedCounter{}, ErrStaleFencingToken
		}
		return ProtectedCounter{}, fmt.Errorf("apply fenced increment: %w", err)
	}

	return counter, nil
}

func (s *Store) GetProtectedCounter(ctx context.Context, resourceName string) (ProtectedCounter, error) {
	const query = `
SELECT resource_name, value, last_fencing_token, last_holder, updated_at
FROM protected_counters
WHERE resource_name = $1`

	var counter ProtectedCounter
	err := s.db.QueryRowContext(ctx, query, resourceName).Scan(
		&counter.ResourceName,
		&counter.Value,
		&counter.LastFencingToken,
		&counter.LastHolder,
		&counter.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProtectedCounter{}, ErrNotFound
		}
		return ProtectedCounter{}, fmt.Errorf("get protected counter: %w", err)
	}

	return counter, nil
}
