package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kurama/auction-system/backend/internal/app"
	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/httputil"
	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/service"
)

type WithdrawalHandler struct {
	withdrawalService *service.WithdrawalService
}

func NewWithdrawalHandler(ctx *app.Context) *WithdrawalHandler {
	queries := repository.New(ctx.DB)
	withdrawalService := service.NewWithdrawalService(ctx.DB, queries, ctx.Hub)

	h := &WithdrawalHandler{withdrawalService: withdrawalService}

	w := ctx.Wrap
	wallet := ctx.Engine.Group("/api/wallet")
	wallet.Use(ctx.Auth)
	{
		wallet.POST("/withdrawal", w(h.CreateWithdrawal))
		wallet.GET("/withdrawals", w(h.ListMyWithdrawals))
	}

	return h
}

func (h *WithdrawalHandler) CreateWithdrawal(c *gin.Context) error {
	var req model.WithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	userID := httputil.GetUserIDFromContext(c)
	withdrawal, err := h.withdrawalService.CreateWithdrawal(c.Request.Context(), userID, req)
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusCreated, c, httputil.NewCreatedResponse(c, withdrawal))
}

func (h *WithdrawalHandler) ListMyWithdrawals(c *gin.Context) error {
	userID := httputil.GetUserIDFromContext(c)
	p := httputil.ParsePaginationFromQuery(c.Request)

	withdrawals, err := h.withdrawalService.ListByUser(c.Request.Context(), userID, int32(p.GetLimit()), int32(p.GetOffset()))
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, withdrawals))
}
