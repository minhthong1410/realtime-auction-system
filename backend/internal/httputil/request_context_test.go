package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestWithRequestContext(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test", nil)
	rc := &RequestContext{UserID: "user-1", Username: "alice"}

	newReq := WithRequestContext(req, rc)

	val := newReq.Context().Value(RequestContextKey)
	assert.NotNil(t, val)
	assert.Equal(t, "user-1", val.(*RequestContext).UserID)
	assert.Equal(t, "alice", val.(*RequestContext).Username)
}

func TestGetRequestContext(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		rc := &RequestContext{UserID: "uid", Username: "uname"}
		c.Request = WithRequestContext(c.Request, rc)

		got, ok := GetRequestContext(c)
		assert.True(t, ok)
		assert.Equal(t, "uid", got.UserID)
		assert.Equal(t, "uname", got.Username)
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestGetRequestContextMissing(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		got, ok := GetRequestContext(c)
		assert.False(t, ok)
		assert.Nil(t, got)
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestGetUserIDFromContext(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Request = WithRequestContext(c.Request, &RequestContext{UserID: "abc-123"})
		assert.Equal(t, "abc-123", GetUserIDFromContext(c))
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestGetUserIDFromContextMissing(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		assert.Equal(t, "", GetUserIDFromContext(c))
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestGetUsernameFromContext(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Request = WithRequestContext(c.Request, &RequestContext{Username: "alice"})
		assert.Equal(t, "alice", GetUsernameFromContext(c))
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestGetUsernameFromContextMissing(t *testing.T) {
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		assert.Equal(t, "", GetUsernameFromContext(c))
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}
