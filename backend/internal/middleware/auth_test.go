package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kurama/auction-system/backend/internal/httputil"
	"github.com/kurama/auction-system/backend/internal/i18n"
	"github.com/kurama/auction-system/backend/internal/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

const testSecret = "test-jwt-secret-for-auth-middleware"

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	l, _ := zap.NewDevelopment()
	logger.Init(l)
	i18n.Init()
	os.Exit(m.Run())
}

func generateTestToken(userID, username, tokenType string, expiry time.Duration, secret string) string {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"type":     tokenType,
		"exp":      time.Now().Add(expiry).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

func setupAuthRouter() *gin.Engine {
	r := gin.New()
	r.Use(AuthMiddleware(testSecret))
	r.GET("/protected", func(c *gin.Context) {
		userID := httputil.GetUserIDFromContext(c)
		username := httputil.GetUsernameFromContext(c)
		c.JSON(200, gin.H{"user_id": userID, "username": username})
	})
	return r
}

func TestAuthMiddlewareValidToken(t *testing.T) {
	r := setupAuthRouter()
	token := generateTestToken("user-123", "alice", "access", 15*time.Minute, testSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "user-123")
	assert.Contains(t, w.Body.String(), "alice")
}

func TestAuthMiddlewareMissingHeader(t *testing.T) {
	r := setupAuthRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
	assert.Contains(t, w.Body.String(), "missing authorization header")
}

func TestAuthMiddlewareInvalidFormat(t *testing.T) {
	r := setupAuthRouter()

	formats := []string{
		"token-without-bearer",
		"Basic dXNlcjpwYXNz",
		"Bearer",     // no token
		"bearer abc", // lowercase bearer
		"",
	}

	for _, header := range formats {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		r.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code, "header %q should be rejected", header)
	}
}

func TestAuthMiddlewareExpiredToken(t *testing.T) {
	r := setupAuthRouter()
	token := generateTestToken("user-123", "alice", "access", -1*time.Hour, testSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired token")
}

func TestAuthMiddlewareWrongSecret(t *testing.T) {
	r := setupAuthRouter()
	token := generateTestToken("user-123", "alice", "access", 15*time.Minute, "wrong-secret")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestAuthMiddlewareMalformedToken(t *testing.T) {
	r := setupAuthRouter()

	tokens := []string{"not.a.jwt", "abc", "eyJ...", "a.b.c"}
	for _, tok := range tokens {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		r.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code, "token %q should be rejected", tok)
	}
}

func TestAuthMiddlewareMissingUserID(t *testing.T) {
	r := setupAuthRouter()

	// Token without user_id claim
	claims := jwt.MapClaims{
		"username": "alice",
		"type":     "access",
		"exp":      time.Now().Add(15 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(testSecret))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
	assert.Contains(t, w.Body.String(), "invalid user_id")
}

func TestAuthMiddlewareNoneAlgorithm(t *testing.T) {
	r := setupAuthRouter()

	// Attempt algorithm confusion attack
	claims := jwt.MapClaims{
		"user_id": "attacker", "type": "access",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestAuthMiddlewareUserIDInContext(t *testing.T) {
	r := gin.New()
	r.Use(AuthMiddleware(testSecret))
	r.GET("/me", func(c *gin.Context) {
		rc, ok := httputil.GetRequestContext(c)
		assert.True(t, ok)
		assert.Equal(t, "ctx-user", rc.UserID)
		assert.Equal(t, "ctx-name", rc.Username)
		c.Status(200)
	})

	token := generateTestToken("ctx-user", "ctx-name", "access", 15*time.Minute, testSecret)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/me", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}
