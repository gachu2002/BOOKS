package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (s *Store) CreateUser(ctx context.Context, params UserCreateParams) (User, error) {
	const query = `
INSERT INTO users (email, full_name, role)
VALUES ($1, $2, $3)
RETURNING id, email, full_name, role, created_at, updated_at`

	var user User
	err := s.db.QueryRowContext(ctx, query, params.Email, params.FullName, params.Role).
		Scan(&user.ID, &user.Email, &user.FullName, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *Store) GetUser(ctx context.Context, id string) (User, error) {
	const query = `
SELECT id, email, full_name, role, created_at, updated_at
FROM users
WHERE id = $1`

	var user User
	err := s.db.QueryRowContext(ctx, query, id).
		Scan(&user.ID, &user.Email, &user.FullName, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("get user: %w", err)
	}

	return user, nil
}

func (s *Store) ListUsers(ctx context.Context, page Pagination) ([]User, error) {
	const query = `
SELECT id, email, full_name, role, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`

	rows, err := s.db.QueryContext(ctx, query, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0, page.Limit)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.FullName, &user.Role, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}

func (s *Store) UpdateUser(ctx context.Context, id string, params UserUpdateParams) (User, error) {
	const query = `
UPDATE users
SET email = COALESCE($2, email),
    full_name = COALESCE($3, full_name),
    role = COALESCE($4, role),
    updated_at = now()
WHERE id = $1
RETURNING id, email, full_name, role, created_at, updated_at`

	var user User
	err := s.db.QueryRowContext(ctx, query, id, params.Email, params.FullName, params.Role).
		Scan(&user.ID, &user.Email, &user.FullName, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("update user: %w", err)
	}

	return user, nil
}

func (s *Store) DeleteUser(ctx context.Context, id string) error {
	const query = `DELETE FROM users WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete user rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
