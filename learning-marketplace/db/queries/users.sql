-- name: CreateUser :one
INSERT INTO users (email, full_name, role)
VALUES ($1, $2, $3)
RETURNING id, email, full_name, role, created_at, updated_at;

-- name: GetUser :one
SELECT id, email, full_name, role, created_at, updated_at
FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, full_name, role, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
UPDATE users
SET email = COALESCE($2, email),
    full_name = COALESCE($3, full_name),
    role = COALESCE($4, role),
    updated_at = now()
WHERE id = $1
RETURNING id, email, full_name, role, created_at, updated_at;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = $1;
