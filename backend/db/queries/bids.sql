-- name: CreateBid :exec
INSERT INTO bids (id, auction_id, user_id, amount)
VALUES (?, ?, ?, ?);

-- name: ListBidsByAuction :many
SELECT b.id, b.auction_id, b.user_id, u.username, b.amount, b.created_at
FROM bids b
JOIN users u ON u.id = b.user_id
WHERE b.auction_id = ?
ORDER BY b.amount DESC
LIMIT ? OFFSET ?;

-- name: CountBidsByAuction :one
SELECT COUNT(*) FROM bids WHERE auction_id = ?;
