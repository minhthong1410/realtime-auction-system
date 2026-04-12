package middleware

import (
	"github.com/gin-gonic/gin"
	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/httputil"
	"github.com/kurama/auction-system/backend/internal/logger"
	"go.uber.org/zap"
)

// HandlerFunc is a gin handler that returns an error.
type HandlerFunc func(c *gin.Context) error

// WrapHandler wraps a HandlerFunc into a gin.HandlerFunc.
func WrapHandler(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h(c); err != nil {
			requestID, _ := c.Get("request_id")
			logger.Error("unhandled handler error",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Any("request_id", requestID),
			)
			if !c.Writer.Written() {
				httputil.RenderError(c, appErr.ErrorInternalServer)
			}
		}
	}
}
