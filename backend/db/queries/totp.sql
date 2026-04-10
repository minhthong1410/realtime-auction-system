-- name: UpdateUserTotpSecret :exec
UPDATE users SET totp_secret = ? WHERE id = ?;

-- name: EnableUserTotp :exec
UPDATE users SET totp_enabled = TRUE, backup_codes = ? WHERE id = ?;

-- name: DisableUserTotp :exec
UPDATE users SET totp_enabled = FALSE, totp_secret = NULL, backup_codes = NULL WHERE id = ?;

-- name: GetUserTotpInfo :one
SELECT id, username, totp_secret, totp_enabled, backup_codes
FROM users WHERE id = ?;

-- name: UpdateUserBackupCodes :exec
UPDATE users SET backup_codes = ? WHERE id = ?;
