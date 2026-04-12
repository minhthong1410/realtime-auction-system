package service

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kurama/auction-system/backend/internal/config"
	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// --- dummyHash ---

func TestDummyHashIsValidBcrypt(t *testing.T) {
	assert.NotNil(t, dummyHash)
	assert.True(t, len(dummyHash) > 0)
	assert.Equal(t, byte('$'), dummyHash[0])
}

func TestDummyHashTimingConsistency(t *testing.T) {
	realHash, _ := bcrypt.GenerateFromPassword([]byte("real-password"), bcrypt.DefaultCost)

	start1 := time.Now()
	bcrypt.CompareHashAndPassword(dummyHash, []byte("wrong"))
	d1 := time.Since(start1)

	start2 := time.Now()
	bcrypt.CompareHashAndPassword(realHash, []byte("wrong"))
	d2 := time.Since(start2)

	ratio := float64(d1) / float64(d2)
	assert.True(t, ratio > 0.2 && ratio < 5.0,
		"timing ratio %f should be near 1 (dummy=%v, real=%v)", ratio, d1, d2)
}

func TestDummyHashNeverMatches(t *testing.T) {
	passwords := []string{"", "password", "123456", "dummy-password-for-timing"}
	for _, p := range passwords {
		// Even the source password shouldn't matter for security —
		// we just check it doesn't accidentally match common passwords
		// (except the actual dummy password which will match)
		if p == "dummy-password-for-timing" {
			continue
		}
		err := bcrypt.CompareHashAndPassword(dummyHash, []byte(p))
		assert.Error(t, err, "dummyHash should not match %q", p)
	}
}

// --- generateTokens ---

func newTestAuthService() *AuthService {
	return &AuthService{
		cfg: config.JWTConfig{
			Secret:          "test-jwt-secret-32-chars-long!!",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
	}
}

func TestGenerateTokens(t *testing.T) {
	s := newTestAuthService()

	user := model.User{
		ID: "test-user-id", Username: "alice",
		Email: "alice@test.com", Balance: 10000,
	}

	resp, err := s.GenerateTokensForUser(user)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotEqual(t, resp.AccessToken, resp.RefreshToken)
	assert.Equal(t, "alice", resp.User.Username)
	assert.Equal(t, int64(10000), resp.User.Balance)
}

func TestGenerateTokensAccessTokenClaims(t *testing.T) {
	s := newTestAuthService()

	resp, _ := s.GenerateTokensForUser(model.User{ID: "uid-1", Username: "bob"})

	token, err := jwt.Parse(resp.AccessToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.Secret), nil
	})
	require.NoError(t, err)

	claims := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "uid-1", claims["user_id"])
	assert.Equal(t, "bob", claims["username"])
	assert.Equal(t, "access", claims["type"])
}

func TestGenerateTokensRefreshTokenClaims(t *testing.T) {
	s := newTestAuthService()

	resp, _ := s.GenerateTokensForUser(model.User{ID: "uid-1", Username: "bob"})

	token, err := jwt.Parse(resp.RefreshToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.Secret), nil
	})
	require.NoError(t, err)

	claims := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "uid-1", claims["user_id"])
	assert.Equal(t, "refresh", claims["type"])
	// refresh token should NOT contain username (security)
	_, hasUsername := claims["username"]
	assert.False(t, hasUsername)
}

func TestGenerateTokensExpiry(t *testing.T) {
	s := newTestAuthService()

	resp, _ := s.GenerateTokensForUser(model.User{ID: "uid-1"})

	// Access token expires in ~15 minutes
	accessToken, _ := jwt.Parse(resp.AccessToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.Secret), nil
	})
	accessExp, _ := accessToken.Claims.(jwt.MapClaims)["exp"].(float64)
	accessTTL := time.Unix(int64(accessExp), 0).Sub(time.Now())
	assert.InDelta(t, 15*60, accessTTL.Seconds(), 5) // within 5 seconds

	// Refresh token expires in ~7 days
	refreshToken, _ := jwt.Parse(resp.RefreshToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.Secret), nil
	})
	refreshExp, _ := refreshToken.Claims.(jwt.MapClaims)["exp"].(float64)
	refreshTTL := time.Unix(int64(refreshExp), 0).Sub(time.Now())
	assert.InDelta(t, 7*24*3600, refreshTTL.Seconds(), 5)
}

// --- userFromRow ---

func TestUserFromRow(t *testing.T) {
	now := time.Now()
	id := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	user := userFromRow(id, "alice", "alice@test.com", 50000, now)

	assert.Equal(t, "01020304-0506-0708-090a-0b0c0d0e0f10", user.ID)
	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, "alice@test.com", user.Email)
	assert.Equal(t, int64(50000), user.Balance)
	assert.Equal(t, now, user.CreatedAt)
}

func TestUserFromRowZeroBalance(t *testing.T) {
	user := userFromRow(make([]byte, 16), "new_user", "new@test.com", 0, time.Now())
	assert.Equal(t, int64(0), user.Balance)
}

func TestUserFromRowNegativeBalance(t *testing.T) {
	// Edge case: negative balance (shouldn't happen but test it doesn't panic)
	user := userFromRow(make([]byte, 16), "edge", "e@t.com", -500, time.Now())
	assert.Equal(t, int64(-500), user.Balance)
}
