package handler

import (
	"os"

	"github.com/gin-gonic/gin"
	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/httputil"
)

func renderServiceError(c *gin.Context, err error) error {
	if cErr, ok := err.(appErr.CustomError); ok {
		return httputil.RenderError(c, cErr)
	}
	return httputil.RenderError(c, appErr.ErrorInternalServer)
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
