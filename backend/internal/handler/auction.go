package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kurama/auction-system/backend/internal/app"
	"github.com/kurama/auction-system/backend/internal/cache"
	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/httputil"
	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/service"
	"github.com/kurama/auction-system/backend/internal/util"
)

type AuctionHandler struct {
	auctionService *service.AuctionService
	bidService     *service.BidService
}

func NewAuctionHandler(ctx *app.Context) *AuctionHandler {
	queries := repository.New(ctx.DB)
	c := cache.New(ctx.Redis)
	auctionService := service.NewAuctionService(ctx.DB, queries, c)
	bidService := service.NewBidService(ctx.DB, queries, ctx.Hub)

	h := &AuctionHandler{
		auctionService: auctionService,
		bidService:     bidService,
	}

	w := ctx.Wrap

	// Public
	public := ctx.Engine.Group("/api/auctions")
	{
		public.GET("", w(h.ListActive))
		public.GET("/:id", w(h.GetByID))
		public.GET("/:id/bids", w(h.GetBidHistory))
	}

	// Protected
	protected := ctx.Engine.Group("/api/auctions")
	protected.Use(ctx.Auth)
	{
		protected.POST("", w(h.Create))
		protected.POST("/:id/bid", w(h.PlaceBid))
	}

	myGroup := ctx.Engine.Group("/api/my")
	myGroup.Use(ctx.Auth)
	{
		myGroup.GET("/auctions", w(h.ListByUser))
	}

	return h
}

func (h *AuctionHandler) Create(c *gin.Context) error {
	var req model.CreateAuctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	userID := httputil.GetUserIDFromContext(c)
	auction, err := h.auctionService.Create(c.Request.Context(), userID, req)
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusCreated, c, httputil.NewCreatedResponse(c, auction))
}

func (h *AuctionHandler) GetByID(c *gin.Context) error {
	id := c.Param("id")
	if !util.IsValidUUID(id) {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	auction, err := h.auctionService.GetByID(c.Request.Context(), id)
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, auction))
}

func (h *AuctionHandler) ListActive(c *gin.Context) error {
	p := httputil.ParsePaginationFromQuery(c.Request)

	auctions, total, err := h.auctionService.ListActive(c.Request.Context(), int32(p.GetLimit()), int32(p.GetOffset()))
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewPaginatedResponse(c, auctions, p.Page, p.Size, total))
}

func (h *AuctionHandler) ListByUser(c *gin.Context) error {
	userID := httputil.GetUserIDFromContext(c)
	p := httputil.ParsePaginationFromQuery(c.Request)

	auctions, err := h.auctionService.ListByUser(c.Request.Context(), userID, int32(p.GetLimit()), int32(p.GetOffset()))
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, auctions))
}

func (h *AuctionHandler) GetBidHistory(c *gin.Context) error {
	auctionID := c.Param("id")
	if !util.IsValidUUID(auctionID) {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	p := httputil.ParsePaginationFromQuery(c.Request)

	bids, err := h.auctionService.GetBidHistory(c.Request.Context(), auctionID, int32(p.GetLimit()), int32(p.GetOffset()))
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, bids))
}

func (h *AuctionHandler) PlaceBid(c *gin.Context) error {
	auctionID := c.Param("id")
	if !util.IsValidUUID(auctionID) {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	var req model.PlaceBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	userID := httputil.GetUserIDFromContext(c)
	username := httputil.GetUsernameFromContext(c)

	bid, err := h.bidService.PlaceBid(c.Request.Context(), auctionID, userID, username, req.Amount)
	if err != nil {
		return renderServiceError(c, err)
	}

	// Invalidate auction cache after successful bid
	h.auctionService.InvalidateAuction(c.Request.Context(), auctionID)

	return httputil.RenderGinJSON(http.StatusCreated, c, httputil.NewCreatedResponse(c, bid))
}
