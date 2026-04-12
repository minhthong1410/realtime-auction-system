-- name: CreateWithdrawal :exec
INSERT INTO withdrawals (id, user_id, amount, status, bank_name, account_number, account_holder, note)
VALUES (?, ?, ?, 'pending', ?, ?, ?, ?);

-- name: GetWithdrawalByID :one
SELECT w.id, w.user_id, u.username, w.amount, w.status, w.bank_name, w.account_number, w.account_holder, w.note, w.reviewed_at, w.created_at, w.updated_at
FROM withdrawals w
JOIN users u ON u.id = w.user_id
WHERE w.id = ?;

-- name: ListWithdrawalsByUser :many
SELECT id, user_id, amount, status, bank_name, account_number, account_holder, note, reviewed_at, created_at, updated_at
FROM withdrawals
WHERE user_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateWithdrawalStatus :exec
UPDATE withdrawals SET status = ?, reviewed_at = NOW() WHERE id = ?;

-- name: CountPendingWithdrawalsByUser :one
SELECT COUNT(*) FROM withdrawals WHERE user_id = ? AND status = 'pending';

-- name: LockUserForWithdrawal :one
SELECT id, balance FROM users WHERE id = ? FOR UPDATE;
