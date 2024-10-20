// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.24.0
// source: user.sql

package db

import (
	"context"
)

const createUser = `-- name: CreateUser :one
INSERT INTO
    users (
        email,
        hashed_password,
        full_name,
        role
    )
VALUES (
        $1, -- email
        $2, -- hash_password
        $3, -- role
        $4 -- full_name
    ) RETURNING email, hashed_password, full_name, role, created_at
`

type CreateUserParams struct {
	Email          string `json:"email"`
	HashedPassword string `json:"hashed_password"`
	FullName       string `json:"full_name"`
	Role           string `json:"role"`
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (Users, error) {
	row := q.db.QueryRowContext(ctx, createUser,
		arg.Email,
		arg.HashedPassword,
		arg.FullName,
		arg.Role,
	)
	var i Users
	err := row.Scan(
		&i.Email,
		&i.HashedPassword,
		&i.FullName,
		&i.Role,
		&i.CreatedAt,
	)
	return i, err
}

const deleteUser = `-- name: DeleteUser :exec
DELETE FROM users WHERE email = $1
`

func (q *Queries) DeleteUser(ctx context.Context, email string) error {
	_, err := q.db.ExecContext(ctx, deleteUser, email)
	return err
}

const getUser = `-- name: GetUser :one
SELECT email, hashed_password, full_name, role, created_at FROM users WHERE email = $1 LIMIT 1
`

func (q *Queries) GetUser(ctx context.Context, email string) (Users, error) {
	row := q.db.QueryRowContext(ctx, getUser, email)
	var i Users
	err := row.Scan(
		&i.Email,
		&i.HashedPassword,
		&i.FullName,
		&i.Role,
		&i.CreatedAt,
	)
	return i, err
}

const updateUser = `-- name: UpdateUser :one
UPDATE users
SET
    full_name = COALESCE($1, full_name),
    hashed_password = COALESCE($2, hashed_password),
    email = COALESCE($3, email),
    role = COALESCE($4, role),
    updated_at = now()
WHERE
    email = $5 RETURNING email, hashed_password, full_name, role, created_at
`

type UpdateUserParams struct {
	FullName       string `json:"full_name"`
	HashedPassword string `json:"hashed_password"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	Email_2        string `json:"email_2"`
}

func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) (Users, error) {
	row := q.db.QueryRowContext(ctx, updateUser,
		arg.FullName,
		arg.HashedPassword,
		arg.Email,
		arg.Role,
		arg.Email_2,
	)
	var i Users
	err := row.Scan(
		&i.Email,
		&i.HashedPassword,
		&i.FullName,
		&i.Role,
		&i.CreatedAt,
	)
	return i, err
}
