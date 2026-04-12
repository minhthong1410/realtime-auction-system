package service

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testJWTSecret = []byte("test-secret-key-for-unit-tests")

func newTestTOTPService() *TOTPService {
	return &TOTPService{
		jwtSecret: testJWTSecret,
		aesKey:    []byte("01234567890123456789012345678901"),
		issuer:    "TestAuction",
	}
}

// --- GenerateTempToken ---

func TestGenerateTempToken(t *testing.T) {
	s := newTestTOTPService()

	token, err := s.GenerateTempToken("user-123", "alice", "totp_setup")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, strings.Count(token, ".") == 2, "should be a JWT with 3 parts")
}

func TestGenerateTempTokenDifferentPurposes(t *testing.T) {
	s := newTestTOTPService()

	t1, _ := s.GenerateTempToken("user-1", "alice", "totp_setup")
	t2, _ := s.GenerateTempToken("user-1", "alice", "totp_verify")
	assert.NotEqual(t, t1, t2, "different purposes should produce different tokens")
}

func TestGenerateTempTokenDifferentUsers(t *testing.T) {
	s := newTestTOTPService()

	t1, _ := s.GenerateTempToken("user-1", "alice", "totp_setup")
	t2, _ := s.GenerateTempToken("user-2", "bob", "totp_setup")
	assert.NotEqual(t, t1, t2)
}

// --- ValidateTempToken ---

func TestValidateTempTokenSuccess(t *testing.T) {
	s := newTestTOTPService()

	token, _ := s.GenerateTempToken("user-123", "alice", "totp_setup")

	claims, err := s.ValidateTempToken(token, "totp_setup")
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "alice", claims.Username)
	assert.Equal(t, "totp_setup", claims.Purpose)
}

func TestValidateTempTokenWrongPurpose(t *testing.T) {
	s := newTestTOTPService()

	token, _ := s.GenerateTempToken("user-123", "alice", "totp_setup")

	_, err := s.ValidateTempToken(token, "totp_verify")
	assert.Error(t, err)
}

func TestValidateTempTokenExpired(t *testing.T) {
	s := newTestTOTPService()

	claims := TempTokenClaims{
		UserID: "user-123", Username: "alice", Purpose: "totp_setup",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString(testJWTSecret)

	_, err := s.ValidateTempToken(tokenStr, "totp_setup")
	assert.Error(t, err)
}

func TestValidateTempTokenInvalidSignature(t *testing.T) {
	s := newTestTOTPService()

	claims := TempTokenClaims{
		UserID: "user-123", Username: "alice", Purpose: "totp_setup",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte("wrong-secret"))

	_, err := s.ValidateTempToken(tokenStr, "totp_setup")
	assert.Error(t, err)
}

func TestValidateTempTokenMalformed(t *testing.T) {
	s := newTestTOTPService()

	malformed := []string{
		"", "not.a.jwt", "a.b.c", "eyJhbGciOiJIUzI1NiJ9.invalid.sig",
		"   ", "Bearer token",
	}

	for _, tok := range malformed {
		t.Run(tok, func(t *testing.T) {
			_, err := s.ValidateTempToken(tok, "totp_setup")
			assert.Error(t, err)
		})
	}
}

func TestValidateTempTokenWrongAlgorithm(t *testing.T) {
	s := newTestTOTPService()

	// Use none algorithm (should be rejected)
	token := jwt.NewWithClaims(jwt.SigningMethodNone, TempTokenClaims{
		UserID: "user-123", Username: "alice", Purpose: "totp_setup",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	})
	tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err := s.ValidateTempToken(tokenStr, "totp_setup")
	assert.Error(t, err)
}

// --- generateRandomCode ---

func TestGenerateRandomCodeNumericOnly(t *testing.T) {
	for i := 0; i < 100; i++ {
		code := generateRandomCode(6)
		assert.Len(t, code, 6)
		for _, c := range code {
			assert.True(t, c >= '0' && c <= '9', "expected digit, got %c", c)
		}
	}
}

func TestGenerateRandomCodeLengths(t *testing.T) {
	for _, l := range []int{1, 4, 6, 8, 10, 20} {
		code := generateRandomCode(l)
		assert.Len(t, code, l)
	}
}

func TestGenerateRandomCodeUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := generateRandomCode(6)
		seen[code] = true
	}
	// With 6 digits (1M possibilities), 100 codes should have very few collisions
	assert.Greater(t, len(seen), 90, "should generate mostly unique codes")
}

// --- Constants ---

func TestTOTPConstants(t *testing.T) {
	assert.Equal(t, "totp_verify", tempTokenPurposeVerify)
	assert.Equal(t, "totp_setup", tempTokenPurposeSetup)
	assert.Equal(t, 15*time.Minute, tempTokenExpiry)
	assert.Equal(t, 5, otpMaxAttempts)
	assert.Equal(t, 15*time.Minute, otpAttemptsTTL)
	assert.Equal(t, 10, backupCodeCount)
	assert.Equal(t, 6, backupCodeLength)
}
