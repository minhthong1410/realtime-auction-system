package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"

	"github.com/kurama/auction-system/backend/internal/config"
	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/logger"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/util"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	tempTokenPurposeVerify = "totp_verify"
	tempTokenPurposeSetup  = "totp_setup"
	tempTokenExpiry        = 15 * time.Minute
	otpMaxAttempts         = 5
	otpAttemptsTTL         = 15 * time.Minute
	backupCodeCount  = 10
	backupCodeLength = 6
)

type TempTokenClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Purpose  string `json:"purpose"`
	jwt.RegisteredClaims
}

type TOTPService struct {
	queries   *repository.Queries
	db        *sql.DB
	rdb       *redis.Client
	aesKey    []byte
	issuer    string
	jwtSecret []byte
}

func NewTOTPService(db *sql.DB, queries *repository.Queries, rdb *redis.Client, cfg config.TOTPConfig, jwtSecret string) *TOTPService {
	return &TOTPService{
		queries:   queries,
		db:        db,
		rdb:       rdb,
		aesKey:    []byte(cfg.AESKey),
		issuer:    cfg.Issuer,
		jwtSecret: []byte(jwtSecret),
	}
}

// GenerateTempToken creates a short-lived JWT for the TOTP flow.
func (s *TOTPService) GenerateTempToken(userID, username, purpose string) (string, error) {
	claims := TempTokenClaims{
		UserID:   userID,
		Username: username,
		Purpose:  purpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tempTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidateTempToken validates a temp token and checks purpose.
func (s *TOTPService) ValidateTempToken(tokenStr, expectedPurpose string) (*TempTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TempTokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, appErr.ErrorTotpTempTokenExpired
	}

	claims, ok := token.Claims.(*TempTokenClaims)
	if !ok || claims.Purpose != expectedPurpose {
		return nil, appErr.ErrorTotpTempTokenExpired
	}

	return claims, nil
}

// SetupTOTP generates a new TOTP secret, stores encrypted, returns QR code.
func (s *TOTPService) SetupTOTP(ctx context.Context, userID, username string) (qrBase64 string, secret string, err error) {
	userIDBytes, err := util.UUIDFromString(userID)
	if err != nil {
		logger.Error("totp setup: invalid user ID format", zap.String("user_id", userID), zap.Error(err))
		return "", "", appErr.ErrorNotFound
	}

	info, err := s.queries.GetUserTotpInfo(ctx, userIDBytes)
	if err != nil {
		logger.Error("totp setup: user not found in DB", zap.String("user_id", userID), zap.Error(err))
		return "", "", appErr.ErrorNotFound
	}
	if info.TotpEnabled {
		logger.Warn("totp setup: already enabled", zap.String("user_id", userID))
		return "", "", appErr.ErrorTotpAlreadyEnabled
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: username,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		logger.Error("totp setup: failed to generate key", zap.String("user_id", userID), zap.Error(err))
		return "", "", appErr.ErrorInternalServer
	}

	encrypted, err := util.Encrypt(key.Secret(), s.aesKey)
	if err != nil {
		logger.Error("totp setup: failed to encrypt secret", zap.String("user_id", userID), zap.Error(err))
		return "", "", appErr.ErrorInternalServer
	}

	err = s.queries.UpdateUserTotpSecret(ctx, repository.UpdateUserTotpSecretParams{
		TotpSecret: sql.NullString{String: encrypted, Valid: true},
		ID:         userIDBytes,
	})
	if err != nil {
		logger.Error("totp setup: failed to store secret", zap.String("user_id", userID), zap.Error(err))
		return "", "", appErr.ErrorDatabase
	}

	// Generate QR code
	qr, err := qrcode.New(key.URL(), qrcode.Medium)
	if err != nil {
		return "", "", appErr.ErrorInternalServer
	}
	img := qr.Image(200)

	// Encode to base64 PNG
	var buf []byte
	writer := &pngWriter{data: &buf}
	if err := png.Encode(writer, img); err != nil {
		return "", "", appErr.ErrorInternalServer
	}

	return base64.StdEncoding.EncodeToString(buf), key.Secret(), nil
}

// ConfirmTOTP verifies the code and enables TOTP, returns backup codes.
func (s *TOTPService) ConfirmTOTP(ctx context.Context, userID, code string) ([]string, error) {
	userIDBytes, err := util.UUIDFromString(userID)
	if err != nil {
		logger.Error("totp confirm: invalid user ID format", zap.String("user_id", userID), zap.Error(err))
		return nil, appErr.ErrorNotFound
	}

	info, err := s.queries.GetUserTotpInfo(ctx, userIDBytes)
	if err != nil {
		logger.Error("totp confirm: user not found in DB", zap.String("user_id", userID), zap.Error(err))
		return nil, appErr.ErrorNotFound
	}

	if info.TotpEnabled {
		return nil, appErr.ErrorTotpAlreadyEnabled
	}

	if !info.TotpSecret.Valid {
		return nil, appErr.ErrorTotpNotEnabled
	}

	secret, err := util.Decrypt(info.TotpSecret.String, s.aesKey)
	if err != nil {
		logger.Error("totp confirm: failed to decrypt secret", zap.String("user_id", userID), zap.Error(err))
		return nil, appErr.ErrorInternalServer
	}

	// Validate code
	if !totp.Validate(code, secret) {
		return nil, appErr.ErrorTotpInvalidCode
	}

	// Generate backup codes
	plainCodes := make([]string, backupCodeCount)
	hashedCodes := make([]string, backupCodeCount)
	for i := 0; i < backupCodeCount; i++ {
		code := generateRandomCode(backupCodeLength)
		plainCodes[i] = code
		hashed, _ := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		hashedCodes[i] = string(hashed)
	}

	codesJSON, _ := json.Marshal(hashedCodes)

	err = s.queries.EnableUserTotp(ctx, repository.EnableUserTotpParams{
		BackupCodes: codesJSON,
		ID:          userIDBytes,
	})
	if err != nil {
		logger.Error("totp confirm: failed to enable", zap.String("user_id", userID), zap.Error(err))
		return nil, appErr.ErrorDatabase
	}

	// Reset attempts
	s.rdb.Del(ctx, fmt.Sprintf("user:otp_attempts:%s", userID))

	return plainCodes, nil
}

// VerifyTOTP verifies a TOTP code or backup code.
func (s *TOTPService) VerifyTOTP(ctx context.Context, userID, code string) error {
	attemptsKey := fmt.Sprintf("user:otp_attempts:%s", userID)
	attempts, _ := s.rdb.Incr(ctx, attemptsKey).Result()
	if attempts == 1 {
		s.rdb.Expire(ctx, attemptsKey, otpAttemptsTTL)
	}
	if attempts > int64(otpMaxAttempts) {
		logger.Warn("totp verify: too many attempts", zap.String("user_id", userID), zap.Int64("attempts", attempts))
		return appErr.ErrorTotpTooManyAttempts
	}

	userIDBytes, err := util.UUIDFromString(userID)
	if err != nil {
		logger.Error("totp verify: invalid user ID format", zap.String("user_id", userID), zap.Error(err))
		return appErr.ErrorNotFound
	}

	info, err := s.queries.GetUserTotpInfo(ctx, userIDBytes)
	if err != nil {
		logger.Error("totp verify: user not found in DB", zap.String("user_id", userID), zap.Error(err))
		return appErr.ErrorNotFound
	}

	if !info.TotpEnabled || !info.TotpSecret.Valid {
		return appErr.ErrorTotpNotEnabled
	}

	secret, err := util.Decrypt(info.TotpSecret.String, s.aesKey)
	if err != nil {
		logger.Error("totp verify: failed to decrypt secret", zap.String("user_id", userID), zap.Error(err))
		return appErr.ErrorInternalServer
	}

	// Try TOTP validation
	valid, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err == nil && valid {
		// Replay protection: reject code if already used within this time window
		usedKey := fmt.Sprintf("user:totp_used:%s:%s", userID, code)
		if s.rdb.SetNX(ctx, usedKey, 1, 90*time.Second).Val() {
			s.rdb.Del(ctx, attemptsKey)
			return nil
		}
		// Code was already used — treat as invalid
		return appErr.ErrorTotpInvalidCode
	}

	// Try backup code
	var hashedCodes []string
	if err := json.Unmarshal(info.BackupCodes, &hashedCodes); err == nil {
		for i, hashed := range hashedCodes {
			if bcrypt.CompareHashAndPassword([]byte(hashed), []byte(code)) == nil {
				// Remove used code
				hashedCodes = append(hashedCodes[:i], hashedCodes[i+1:]...)
				updatedJSON, _ := json.Marshal(hashedCodes)
				s.queries.UpdateUserBackupCodes(ctx, repository.UpdateUserBackupCodesParams{
					BackupCodes: updatedJSON,
					ID:          userIDBytes,
				})
				s.rdb.Del(ctx, attemptsKey)
				return nil
			}
		}
	}

	return appErr.ErrorTotpInvalidCode
}

// DisableTOTP removes TOTP from user account.
func (s *TOTPService) DisableTOTP(ctx context.Context, userID string) error {
	userIDBytes, err := util.UUIDFromString(userID)
	if err != nil {
		logger.Error("totp disable: invalid user ID format", zap.String("user_id", userID), zap.Error(err))
		return appErr.ErrorNotFound
	}
	if err := s.queries.DisableUserTotp(ctx, userIDBytes); err != nil {
		logger.Error("totp disable: failed", zap.String("user_id", userID), zap.Error(err))
		return appErr.ErrorDatabase
	}
	return nil
}

// IsTOTPEnabled checks if user has TOTP enabled.
func (s *TOTPService) IsTOTPEnabled(ctx context.Context, userID string) (bool, error) {
	userIDBytes, err := util.UUIDFromString(userID)
	if err != nil {
		logger.Error("totp check enabled: invalid user ID format", zap.String("user_id", userID), zap.Error(err))
		return false, err
	}
	info, err := s.queries.GetUserTotpInfo(ctx, userIDBytes)
	if err != nil {
		logger.Error("totp check enabled: user not found", zap.String("user_id", userID), zap.Error(err))
		return false, err
	}
	return info.TotpEnabled, nil
}

func generateRandomCode(length int) string {
	const charset = "0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

// pngWriter implements io.Writer to collect PNG bytes.
type pngWriter struct {
	data *[]byte
}

func (w *pngWriter) Write(p []byte) (n int, err error) {
	*w.data = append(*w.data, p...)
	return len(p), nil
}
