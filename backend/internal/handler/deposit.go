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

type DepositHandler struct {
	depositService *service.DepositService
}

func NewDepositHandler(ctx *app.Context) *DepositHandler {
	queries := repository.New(ctx.DB)
	frontendURL := getEnvOrDefault("FRONTEND_URL", "http://localhost:3000")
	depositService := service.NewDepositService(ctx.DB, queries, ctx.Hub, frontendURL)

	h := &DepositHandler{depositService: depositService}

	w := ctx.Wrap
	wallet := ctx.Engine.Group("/api/wallet")
	wallet.Use(ctx.Auth)
	{
		wallet.POST("/deposit", w(h.CreateDeposit))
		wallet.GET("/deposits", w(h.ListMyDeposits))
	}

	return h
}

func (h *DepositHandler) CreateDeposit(c *gin.Context) error {
	var req model.DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	userID := httputil.GetUserIDFromContext(c)
	resp, err := h.depositService.CreateDeposit(c.Request.Context(), userID, req.Amount)
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusCreated, c, httputil.NewCreatedResponse(c, resp))
}

func (h *DepositHandler) ListMyDeposits(c *gin.Context) error {
	userID := httputil.GetUserIDFromContext(c)
	p := httputil.ParsePaginationFromQuery(c.Request)

	deposits, err := h.depositService.ListByUser(c.Request.Context(), userID, int32(p.GetLimit()), int32(p.GetOffset()))
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, deposits))
}
