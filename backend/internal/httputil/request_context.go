package httputil

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RequestContext struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

const RequestContextKey = "request-context"

func GetRequestContext(c *gin.Context) (*RequestContext, bool) {
	val := c.Request.Context().Value(RequestContextKey)
	if rc, ok := val.(*RequestContext); ok {
		return rc, true
	}
	return nil, false
}

func GetUserIDFromContext(c *gin.Context) string {
	rc, ok := GetRequestContext(c)
	if !ok {
		return ""
	}
	return rc.UserID
}

func GetUsernameFromContext(c *gin.Context) string {
	rc, ok := GetRequestContext(c)
	if !ok {
		return ""
	}
	return rc.Username
}

func WithRequestContext(r *http.Request, reqCtx *RequestContext) *http.Request {
	return r.WithContext(
		context.WithValue(r.Context(), RequestContextKey, reqCtx),
	)
}
