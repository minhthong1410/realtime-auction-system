-- name: CreateUser :exec
INSERT INTO users (id, username, email, password, balance)
VALUES (?, ?, ?, ?, 0);

-- name: GetUserByEmail :one
SELECT id, username, email, password, balance, created_at
FROM users WHERE email = ?;

-- name: GetUserByID :one
SELECT id, username, email, password, balance, created_at
FROM users WHERE id = ?;

-- name: GetUserByUsername :one
SELECT id, username, email, password, balance, created_at
FROM users WHERE username = ?;

-- name: DepositBalance :execresult
UPDATE users SET balance = balance + ? WHERE id = ?;

-- name: GetBalance :one
SELECT balance FROM users WHERE id = ?;

-- name: DeductBalance :execresult
UPDATE users SET balance = balance - ? WHERE id = ? AND balance >= ?;

-- name: RefundBalance :exec
UPDATE users SET balance = balance + ? WHERE id = ?;
