ALTER TABLE users
    ADD COLUMN totp_secret   TEXT NULL,
    ADD COLUMN totp_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN backup_codes  JSON NULL;
