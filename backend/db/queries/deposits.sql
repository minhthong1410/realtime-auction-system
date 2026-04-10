-- name: CreateDeposit :exec
INSERT INTO deposits (id, user_id, amount, status, stripe_payment_id)
VALUES (?, ?, ?, 'pending', ?);

-- name: GetDepositByID :one
SELECT d.id, d.user_id, u.username, d.amount, d.status, d.stripe_payment_id, d.created_at, d.updated_at
FROM deposits d
JOIN users u ON u.id = d.user_id
WHERE d.id = ?;

-- name: GetDepositByStripeID :one
SELECT d.id, d.user_id, u.username, d.amount, d.status, d.stripe_payment_id, d.created_at, d.updated_at
FROM deposits d
JOIN users u ON u.id = d.user_id
WHERE d.stripe_payment_id = ?;

-- name: UpdateDepositStatus :exec
UPDATE deposits SET status = ? WHERE id = ?;

-- name: ListDepositsByUser :many
SELECT id, user_id, amount, status, stripe_payment_id, created_at, updated_at
FROM deposits
WHERE user_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;
