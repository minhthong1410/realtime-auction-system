CREATE TABLE withdrawals (
    id              BINARY(16) NOT NULL PRIMARY KEY,
    user_id         BINARY(16) NOT NULL,
    amount          BIGINT NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    bank_name       VARCHAR(100) NOT NULL,
    account_number  VARCHAR(50) NOT NULL,
    account_holder  VARCHAR(100) NOT NULL,
    note            VARCHAR(255) NOT NULL DEFAULT '',
    reviewed_at     TIMESTAMP NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    INDEX idx_withdrawals_user (user_id),
    INDEX idx_withdrawals_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
