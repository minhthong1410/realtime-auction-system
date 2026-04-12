-- name: CreateAuction :exec
INSERT INTO auctions (id, seller_id, title, description, images, starting_price, current_price, status, start_time, end_time)
VALUES (?, ?, ?, ?, ?, ?, ?, 1, NOW(), ?);

-- name: GetAuctionByID :one
SELECT a.id, a.seller_id, u.username as seller_name,
       a.title, a.description, a.images,
       a.starting_price, a.current_price, a.winner_id,
       COALESCE(w.username, '') as winner_name,
       a.status, a.start_time, a.end_time, a.created_at,
       (SELECT COUNT(*) FROM bids WHERE auction_id = a.id) as bid_count
FROM auctions a
JOIN users u ON u.id = a.seller_id
LEFT JOIN users w ON w.id = a.winner_id
WHERE a.id = ?;

-- name: ListActiveAuctions :many
SELECT a.id, a.seller_id, u.username as seller_name,
       a.title, a.description, a.images,
       a.starting_price, a.current_price, a.winner_id,
       a.status, a.start_time, a.end_time, a.created_at,
       (SELECT COUNT(*) FROM bids WHERE auction_id = a.id) as bid_count
FROM auctions a
JOIN users u ON u.id = a.seller_id
WHERE a.status = 1
ORDER BY a.end_time ASC
LIMIT ? OFFSET ?;

-- name: LockAuctionForBid :one
SELECT id, seller_id, current_price, winner_id, status, end_time
FROM auctions
WHERE id = ? AND status = 1
FOR UPDATE;

-- name: UpdateAuctionBid :exec
UPDATE auctions
SET current_price = ?, winner_id = ?
WHERE id = ?;

-- name: GetExpiredActiveAuctions :many
SELECT id, seller_id, winner_id, current_price, title
FROM auctions
WHERE status = 1 AND end_time <= NOW();

-- name: CloseAuctionByID :exec
UPDATE auctions SET status = 2 WHERE id = ?;

-- name: ListAuctionsByUser :many
SELECT a.id, a.seller_id, u.username as seller_name,
       a.title, a.description, a.images,
       a.starting_price, a.current_price, a.winner_id,
       a.status, a.start_time, a.end_time, a.created_at,
       (SELECT COUNT(*) FROM bids WHERE auction_id = a.id) as bid_count
FROM auctions a
JOIN users u ON u.id = a.seller_id
WHERE a.seller_id = ?
ORDER BY a.created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateAuction :exec
UPDATE auctions
SET title = ?, description = ?, images = ?, end_time = ?
WHERE id = ?;

-- name: GetAuctionBidCount :one
SELECT COUNT(*) FROM bids WHERE auction_id = ?;

-- name: GetAuctionOwner :one
SELECT seller_id, status FROM auctions WHERE id = ?;

-- name: CountActiveAuctions :one
SELECT COUNT(*) FROM auctions WHERE status = 1;
