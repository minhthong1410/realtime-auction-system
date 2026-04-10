package httputil

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/i18n"
)

type BaseResponse struct {
	Code       int         `json:"code"`
	Success    bool        `json:"success"`
	Message    string      `json:"message,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	Timestamp  time.Time   `json:"timestamp,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Size       int   `json:"size"`
	TotalPages int64 `json:"totalPages"`
}

func (r *BaseResponse) Error() string {
	return r.Message
}

func NewSuccessResponse(c *gin.Context, data interface{}) *BaseResponse {
	return &BaseResponse{
		Code:      http.StatusOK,
		Success:   true,
		Message:   i18n.T(c, "OK", nil),
		Data:      data,
		Timestamp: time.Now(),
	}
}

func NewCreatedResponse(c *gin.Context, data interface{}) *BaseResponse {
	return &BaseResponse{
		Code:      http.StatusCreated,
		Success:   true,
		Message:   i18n.T(c, "Created", nil),
		Data:      data,
		Timestamp: time.Now(),
	}
}

func NewPaginatedResponse(c *gin.Context, data interface{}, page, pageSize int, total int64) *BaseResponse {
	totalPages := total / int64(pageSize)
	if total%int64(pageSize) > 0 {
		totalPages++
	}
	return &BaseResponse{
		Code:      http.StatusOK,
		Success:   true,
		Message:   i18n.T(c, "OK", nil),
		Data:      data,
		Timestamp: time.Now(),
		Pagination: &Pagination{
			Total:      total,
			Page:       page,
			Size:       pageSize,
			TotalPages: totalPages,
		},
	}
}

func NewErrorResponse(c *gin.Context, code int, message string) *BaseResponse {
	return &BaseResponse{
		Code:      code,
		Success:   false,
		Message:   i18n.T(c, message, nil),
		Timestamp: time.Now(),
	}
}

// ErrorResponse creates a BaseResponse from a CustomError with i18n translation.
func ErrorResponse(c *gin.Context, cError appErr.CustomError) *BaseResponse {
	return &BaseResponse{
		Code:      cError.StatusCode,
		Success:   false,
		Message:   i18n.T(c, cError.Error(), nil),
		Timestamp: time.Now(),
	}
}

// RenderError is a shorthand for rendering a CustomError response.
func RenderError(c *gin.Context, cError appErr.CustomError) error {
	return RenderGinJSON(cError.StatusCode, c, ErrorResponse(c, cError))
}
