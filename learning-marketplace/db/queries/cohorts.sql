-- name: CreateCohort :one
INSERT INTO cohorts (
    product_id, slug, title, capacity, status, starts_at, ends_at, enrollment_opens_at, enrollment_closes_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, product_id, slug, title, capacity, status, starts_at, ends_at, enrollment_opens_at, enrollment_closes_at, created_at, updated_at;

-- name: GetCohort :one
SELECT id, product_id, slug, title, capacity, status, starts_at, ends_at, enrollment_opens_at, enrollment_closes_at, created_at, updated_at
FROM cohorts
WHERE id = $1;

-- name: ListCohorts :many
SELECT id, product_id, slug, title, capacity, status, starts_at, ends_at, enrollment_opens_at, enrollment_closes_at, created_at, updated_at
FROM cohorts
ORDER BY starts_at ASC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateCohort :one
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
RETURNING id, product_id, slug, title, capacity, status, starts_at, ends_at, enrollment_opens_at, enrollment_closes_at, created_at, updated_at;

-- name: DeleteCohort :execrows
DELETE FROM cohorts WHERE id = $1;
