CREATE TABLE auctions (
    id              BINARY(16) NOT NULL PRIMARY KEY,
    seller_id       BINARY(16) NOT NULL,
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    image_url       TEXT,
    starting_price  BIGINT NOT NULL,
    current_price   BIGINT NOT NULL,
    winner_id       BINARY(16) NULL,
    status          SMALLINT NOT NULL DEFAULT 1,
    start_time      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    end_time        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (seller_id) REFERENCES users(id),
    FOREIGN KEY (winner_id) REFERENCES users(id),
    INDEX idx_auctions_status_end (status, end_time),
    INDEX idx_auctions_seller (seller_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
