package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter(rate int, window time.Duration) *gin.Engine {
	r := gin.New()
	r.Use(RateLimit(rate, window))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })
	return r
}

func doRequest(r *gin.Engine, ip string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = ip + ":1234"
	r.ServeHTTP(w, req)
	return w
}

func TestRateLimitAllowsWithinLimit(t *testing.T) {
	r := setupRouter(5, time.Minute)

	for i := 0; i < 5; i++ {
		w := doRequest(r, "1.2.3.4")
		assert.Equal(t, 200, w.Code, "request %d should pass", i+1)
	}
}

func TestRateLimitBlocksExcess(t *testing.T) {
	r := setupRouter(3, time.Minute)

	for i := 0; i < 3; i++ {
		w := doRequest(r, "5.6.7.8")
		assert.Equal(t, 200, w.Code)
	}

	w := doRequest(r, "5.6.7.8")
	assert.Equal(t, 429, w.Code)
	assert.Contains(t, w.Body.String(), "rate limit exceeded")
}

func TestRateLimitBlocksAll429After(t *testing.T) {
	r := setupRouter(2, time.Minute)

	doRequest(r, "9.9.9.9")
	doRequest(r, "9.9.9.9")

	// All subsequent requests should be 429
	for i := 0; i < 5; i++ {
		w := doRequest(r, "9.9.9.9")
		assert.Equal(t, 429, w.Code)
	}
}

func TestRateLimitDifferentIPsIndependent(t *testing.T) {
	r := setupRouter(2, time.Minute)

	// IP1 exhausts quota
	doRequest(r, "10.0.0.1")
	doRequest(r, "10.0.0.1")
	w := doRequest(r, "10.0.0.1")
	assert.Equal(t, 429, w.Code)

	// IP2 still has full quota
	w = doRequest(r, "10.0.0.2")
	assert.Equal(t, 200, w.Code)
	w = doRequest(r, "10.0.0.2")
	assert.Equal(t, 200, w.Code)
}

func TestRateLimitWindowReset(t *testing.T) {
	r := setupRouter(2, 100*time.Millisecond)

	doRequest(r, "11.11.11.11")
	doRequest(r, "11.11.11.11")
	w := doRequest(r, "11.11.11.11")
	assert.Equal(t, 429, w.Code)

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	w = doRequest(r, "11.11.11.11")
	assert.Equal(t, 200, w.Code, "should reset after window")
}

func TestRateLimitSingleRequest(t *testing.T) {
	r := setupRouter(1, time.Minute)

	w := doRequest(r, "20.20.20.20")
	assert.Equal(t, 200, w.Code)

	w = doRequest(r, "20.20.20.20")
	assert.Equal(t, 429, w.Code)
}

func TestRateLimitZeroRate(t *testing.T) {
	r := setupRouter(0, time.Minute)

	// Rate=0 means all requests are blocked after the first free one
	w := doRequest(r, "30.30.30.30")
	assert.Equal(t, 200, w.Code) // first request creates visitor with count=1

	w = doRequest(r, "30.30.30.30")
	assert.Equal(t, 429, w.Code) // count=2 > rate=0
}
