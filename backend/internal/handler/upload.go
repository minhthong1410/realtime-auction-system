package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kurama/auction-system/backend/internal/app"
	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/httputil"
)

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

const maxUploadSize = 5 << 20 // 5MB

type UploadHandler struct {
	ctx *app.Context
}

func NewUploadHandler(ctx *app.Context) *UploadHandler {
	h := &UploadHandler{ctx: ctx}

	w := ctx.Wrap
	upload := ctx.Engine.Group("/api/upload")
	upload.Use(ctx.Auth)
	{
		upload.POST("/image", w(h.UploadImage))
	}

	return h
}

func (h *UploadHandler) UploadImage(c *gin.Context) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !allowedImageTypes[contentType] {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	ext := filepath.Ext(header.Filename)
	safeName := strings.TrimSuffix(header.Filename, ext)
	safeName = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, safeName)

	url, err := h.ctx.S3.Upload(c.Request.Context(), "auctions", safeName+ext, contentType, file)
	if err != nil {
		return httputil.RenderError(c, appErr.ErrorInternalServer)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, gin.H{"url": url}))
}
