package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func (s *Store) CreateCohort(ctx context.Context, params CohortCreateParams) (Cohort, error) {
	const query = `
INSERT INTO cohorts (
    product_id, slug, title, capacity, status, starts_at, ends_at, enrollment_opens_at, enrollment_closes_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, product_id, slug, title, capacity, status, starts_at, ends_at, enrollment_opens_at, enrollment_closes_at, created_at, updated_at`

	var cohort Cohort
	err := s.db.QueryRowContext(ctx, query,
		params.ProductID,
		params.Slug,
		params.Title,
		params.Capacity,
		params.Status,
		params.StartsAt,
		params.EndsAt,
		params.EnrollmentOpensAt,
		params.EnrollmentClosesAt,
	).Scan(
		&cohort.ID,
		&cohort.ProductID,
		&cohort.Slug,
		&cohort.Title,
		&cohort.Capacity,
		&cohort.Status,
		&cohort.StartsAt,
		&cohort.EndsAt,
		&cohort.EnrollmentOpensAt,
		&cohort.EnrollmentClosesAt,
		&cohort.CreatedAt,
		&cohort.UpdatedAt,
	)
	if err != nil {
		return Cohort{}, fmt.Errorf("create cohort: %w", err)
	}

	return cohort, nil
}

func (s *Store) GetCohort(ctx context.Context, id string) (Cohort, error) {
	const query = `
SELECT id, product_id, slug, title, capacity, status, starts_at, ends_at, enrollment_opens_at, enrollment_closes_at, created_at, updated_at
FROM cohorts
WHERE id = $1`

	var cohort Cohort
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&cohort.ID,
		&cohort.ProductID,
		&cohort.Slug,
		&cohort.Title,
		&cohort.Capacity,
		&cohort.Status,
		&cohort.StartsAt,
		&cohort.EndsAt,
		&cohort.EnrollmentOpensAt,
		&cohort.EnrollmentClosesAt,
		&cohort.CreatedAt,
		&cohort.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Cohort{}, ErrNotFound
		}
		return Cohort{}, fmt.Errorf("get cohort: %w", err)
	}

	return cohort, nil
}

func (s *Store) ListCohorts(ctx context.Context, page Pagination) ([]Cohort, error) {
	const query = `
SELECT id, product_id, slug, title, capacity, status, starts_at, ends_at, enrollment_opens_at, enrollment_closes_at, created_at, updated_at
FROM cohorts
ORDER BY starts_at ASC, created_at DESC
LIMIT $1 OFFSET $2`

	rows, err := s.db.QueryContext(ctx, query, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list cohorts: %w", err)
	}
	defer rows.Close()

	cohorts := make([]Cohort, 0, page.Limit)
	for rows.Next() {
		var cohort Cohort
		if err := rows.Scan(
			&cohort.ID,
			&cohort.ProductID,
			&cohort.Slug,
			&cohort.Title,
			&cohort.Capacity,
			&cohort.Status,
			&cohort.StartsAt,
			&cohort.EndsAt,
			&cohort.EnrollmentOpensAt,
			&cohort.EnrollmentClosesAt,
			&cohort.CreatedAt,
			&cohort.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan cohort: %w", err)
		}
		cohorts = append(cohorts, cohort)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cohorts: %w", err)
	}

	return cohorts, nil
}

func (s *Store) UpdateCohort(ctx context.Context, id string, params CohortUpdateParams) (Cohort, error) {
	const query = `
UPDATE cohorts
SET product_id = COALESCE($2, product_id),
    slug = COALESCE($3, slug),
    title = COALESCE($4, title),
    capacity = COALESCE($5, capacity),
    status = COALESCE($6, status),
    starts_at = COALESCE($7, starts_at),
    ends_at = COALESCE($8, ends_at),
    enrollment_opens_at = CASE WHEN $9 THEN $10 ELSE enrollment_opens_at END,
    enrollment_closes_at = CASE WHEN $11 THEN $12 ELSE enrollment_closes_at END,
    updated_at = now()
WHERE id = $1
RETURNING id, product_id, slug, title, capacity, status, starts_at, ends_at, enrollment_opens_at, enrollment_closes_at, created_at, updated_at`

	var cohort Cohort
	err := s.db.QueryRowContext(ctx, query,
		id,
		params.ProductID,
		params.Slug,
		params.Title,
		params.Capacity,
		params.Status,
		params.StartsAt,
		params.EndsAt,
		params.UpdateEnrollOpen,
		nullTimeArg(params.EnrollmentOpensAt),
		params.UpdateEnrollClose,
		nullTimeArg(params.EnrollmentClosesAt),
	).Scan(
		&cohort.ID,
		&cohort.ProductID,
		&cohort.Slug,
		&cohort.Title,
		&cohort.Capacity,
		&cohort.Status,
		&cohort.StartsAt,
		&cohort.EndsAt,
		&cohort.EnrollmentOpensAt,
		&cohort.EnrollmentClosesAt,
		&cohort.CreatedAt,
		&cohort.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Cohort{}, ErrNotFound
		}
		return Cohort{}, fmt.Errorf("update cohort: %w", err)
	}

	return cohort, nil
}

func (s *Store) DeleteCohort(ctx context.Context, id string) error {
	const query = `DELETE FROM cohorts WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete cohort: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete cohort rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func nullTimeArg(value sql.NullTime) any {
	if !value.Valid {
		return nil
	}

	return value.Time
}

func nullableTimePtr(value time.Time) *time.Time {
	return &value
}
