package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/httputil"
)

// HandlerFunc is a gin handler that returns an error.
type HandlerFunc func(c *gin.Context) error

// WrapHandler wraps a HandlerFunc into a gin.HandlerFunc.
func WrapHandler(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h(c); err != nil {
			requestID, _ := c.Get("request_id")
			slog.Error("unhandled handler error",
				"error", err,
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"request_id", requestID,
			)
			if !c.Writer.Written() {
				httputil.RenderError(c, appErr.ErrorInternalServer)
			}
		}
	}
}
