CREATE TABLE deposits (
    id                  BINARY(16) NOT NULL PRIMARY KEY,
    user_id             BINARY(16) NOT NULL,
    amount              BIGINT NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending',
    stripe_payment_id   VARCHAR(255) NOT NULL,
    created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    INDEX idx_deposits_user (user_id),
    INDEX idx_deposits_status (status),
    INDEX idx_deposits_stripe (stripe_payment_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
