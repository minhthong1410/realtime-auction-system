CREATE TABLE bids (
    id          BINARY(16) NOT NULL PRIMARY KEY,
    auction_id  BINARY(16) NOT NULL,
    user_id     BINARY(16) NOT NULL,
    amount      BIGINT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (auction_id) REFERENCES auctions(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    INDEX idx_bids_auction (auction_id, amount DESC),
    INDEX idx_bids_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
