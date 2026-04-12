package service

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"time"

	"github.com/kurama/auction-system/backend/internal/cache"
	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/metrics"
	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/util"
	"github.com/kurama/auction-system/backend/internal/ws"
)

const (
	auctionDetailTTL = 30 * time.Second
	auctionListTTL   = 15 * time.Second
	auctionFeedRoom  = "auction:feed"
)

type AuctionService struct {
	queries *repository.Queries
	db      *sql.DB
	cache   *cache.Cache
	hub     *ws.Hub
}

func NewAuctionService(db *sql.DB, queries *repository.Queries, c *cache.Cache, hub *ws.Hub) *AuctionService {
	return &AuctionService{queries: queries, db: db, cache: c, hub: hub}
}

func (s *AuctionService) Create(ctx context.Context, userID string, req model.CreateAuctionRequest) (*model.Auction, error) {
	if req.EndTime.Before(time.Now().Add(time.Minute)) {
		return nil, appErr.ErrorInvalidEndTime
	}

	// Backward compat: if image_url is set but images is empty, use image_url
	images := req.Images
	if len(images) == 0 && req.ImageURL != "" {
		images = []string{req.ImageURL}
	}
	if len(images) > 5 {
		return nil, appErr.ErrorInvalidParams
	}

	imagesJSON, _ := json.Marshal(images)

	userIDBytes, _ := util.UUIDFromString(userID)
	auctionID := util.NewUUID()

	err := s.queries.CreateAuction(ctx, repository.CreateAuctionParams{
		ID:            auctionID,
		SellerID:      userIDBytes,
		Title:         req.Title,
		Description:   sql.NullString{String: req.Description, Valid: req.Description != ""},
		Images:        imagesJSON,
		StartingPrice: req.StartingPrice,
		CurrentPrice:  req.StartingPrice,
		EndTime:       req.EndTime,
	})
	if err != nil {
		return nil, appErr.ErrorDatabase
	}

	// Invalidate list cache on new auction
	s.cache.DelPattern(ctx, "cache:auctions:*")

	auction, err := s.GetByID(ctx, util.UUIDToString(auctionID))
	if err != nil {
		return nil, err
	}

	s.hub.BroadcastToRoom(ctx, auctionFeedRoom, model.WSMessage{
		Type: "auction_created",
		Data: auction,
	})

	return auction, nil
}

func (s *AuctionService) Update(ctx context.Context, auctionID, userID string, req model.UpdateAuctionRequest) (*model.Auction, error) {
	idBytes, err := util.UUIDFromString(auctionID)
	if err != nil {
		return nil, appErr.ErrorAuctionNotFound
	}

	// Check ownership and status
	owner, err := s.queries.GetAuctionOwner(ctx, idBytes)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, appErr.ErrorAuctionNotFound
		}
		return nil, appErr.ErrorDatabase
	}

	userIDBytes, _ := util.UUIDFromString(userID)
	if string(owner.SellerID) != string(userIDBytes) {
		return nil, appErr.ErrorNotAuctionOwner
	}
	if owner.Status != int16(model.AuctionStatusActive) {
		return nil, appErr.ErrorAuctionEnded
	}

	if req.EndTime.Before(time.Now().Add(time.Minute)) {
		return nil, appErr.ErrorInvalidEndTime
	}
	if len(req.Images) > 5 {
		return nil, appErr.ErrorInvalidParams
	}

	imagesJSON, _ := json.Marshal(req.Images)

	err = s.queries.UpdateAuction(ctx, repository.UpdateAuctionParams{
		Title:       req.Title,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		Images:      imagesJSON,
		EndTime:     req.EndTime,
		ID:          idBytes,
	})
	if err != nil {
		return nil, appErr.ErrorDatabase
	}

	s.InvalidateAuction(ctx, auctionID)

	auction, err := s.GetByID(ctx, auctionID)
	if err != nil {
		return nil, err
	}

	s.hub.BroadcastToRoom(ctx, auctionFeedRoom, model.WSMessage{
		Type: "auction_updated",
		Data: auction,
	})
	s.hub.BroadcastToRoom(ctx, "auction:"+auctionID, model.WSMessage{
		Type: "auction_updated",
		Data: auction,
	})

	return auction, nil
}

func (s *AuctionService) GetByID(ctx context.Context, id string) (*model.Auction, error) {
	idBytes, err := util.UUIDFromString(id)
	if err != nil {
		return nil, appErr.ErrorAuctionNotFound
	}

	// Try cache
	cacheKey := cache.KeyAuction(id)
	var cached model.Auction
	if s.cache.Get(ctx, cacheKey, &cached) {
		metrics.CacheHits.WithLabelValues("hit").Inc()
		return &cached, nil
	}
	metrics.CacheHits.WithLabelValues("miss").Inc()

	row, err := s.queries.GetAuctionByID(ctx, idBytes)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, appErr.ErrorAuctionNotFound
		}
		return nil, appErr.ErrorDatabase
	}

	auction := mapAuctionDetailRow(row)
	s.cache.Set(ctx, cacheKey, auction, auctionDetailTTL)
	return auction, nil
}

func (s *AuctionService) ListActive(ctx context.Context, limit, offset int32) ([]model.Auction, int64, error) {
	// Try cache
	cacheKey := cache.KeyAuctionList(limit, offset)
	type cachedList struct {
		Auctions []model.Auction `json:"auctions"`
		Count    int64           `json:"count"`
	}
	var cached cachedList
	if s.cache.Get(ctx, cacheKey, &cached) {
		metrics.CacheHits.WithLabelValues("hit").Inc()
		return cached.Auctions, cached.Count, nil
	}
	metrics.CacheHits.WithLabelValues("miss").Inc()

	rows, err := s.queries.ListActiveAuctions(ctx, repository.ListActiveAuctionsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, appErr.ErrorDatabase
	}

	count, err := s.queries.CountActiveAuctions(ctx)
	if err != nil {
		return nil, 0, appErr.ErrorDatabase
	}

	auctions := make([]model.Auction, len(rows))
	for i, row := range rows {
		images := parseImages(row.Images)
		auctions[i] = model.Auction{
			ID:            util.UUIDToString(row.ID),
			SellerID:      util.UUIDToString(row.SellerID),
			SellerName:    row.SellerName,
			Title:         row.Title,
			Description:   row.Description.String,
			Images:        images,
			ImageURL:      firstImage(images),
			StartingPrice: row.StartingPrice,
			CurrentPrice:  row.CurrentPrice,
			WinnerID:      nullBytesToStringPtr(row.WinnerID),
			Status:        model.AuctionStatus(row.Status),
			StartTime:     row.StartTime,
			EndTime:       row.EndTime,
			CreatedAt:     row.CreatedAt,
			BidCount:      int(row.BidCount),
		}
	}

	s.cache.Set(ctx, cacheKey, cachedList{Auctions: auctions, Count: count}, auctionListTTL)
	return auctions, count, nil
}

func (s *AuctionService) ListByUser(ctx context.Context, userID string, limit, offset int32) ([]model.Auction, error) {
	userIDBytes, _ := util.UUIDFromString(userID)
	rows, err := s.queries.ListAuctionsByUser(ctx, repository.ListAuctionsByUserParams{
		SellerID: userIDBytes,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, appErr.ErrorDatabase
	}

	auctions := make([]model.Auction, len(rows))
	for i, row := range rows {
		images := parseImages(row.Images)
		auctions[i] = model.Auction{
			ID:            util.UUIDToString(row.ID),
			SellerID:      util.UUIDToString(row.SellerID),
			SellerName:    row.SellerName,
			Title:         row.Title,
			Description:   row.Description.String,
			Images:        images,
			ImageURL:      firstImage(images),
			StartingPrice: row.StartingPrice,
			CurrentPrice:  row.CurrentPrice,
			WinnerID:      nullBytesToStringPtr(row.WinnerID),
			Status:        model.AuctionStatus(row.Status),
			StartTime:     row.StartTime,
			EndTime:       row.EndTime,
			CreatedAt:     row.CreatedAt,
			BidCount:      int(row.BidCount),
		}
	}

	return auctions, nil
}

func (s *AuctionService) GetBidHistory(ctx context.Context, auctionID string, limit, offset int32) ([]model.Bid, error) {
	idBytes, _ := util.UUIDFromString(auctionID)
	rows, err := s.queries.ListBidsByAuction(ctx, repository.ListBidsByAuctionParams{
		AuctionID: idBytes,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, appErr.ErrorDatabase
	}

	bids := make([]model.Bid, len(rows))
	for i, row := range rows {
		bids[i] = model.Bid{
			ID:        util.UUIDToString(row.ID),
			AuctionID: util.UUIDToString(row.AuctionID),
			UserID:    util.UUIDToString(row.UserID),
			Username:  row.Username,
			Amount:    row.Amount,
			CreatedAt: row.CreatedAt,
		}
	}

	return bids, nil
}

// InvalidateAuction clears cache for a specific auction and list caches.
func (s *AuctionService) InvalidateAuction(ctx context.Context, auctionID string) {
	s.cache.Del(ctx, cache.KeyAuction(auctionID))
	s.cache.DelPattern(ctx, "cache:auctions:list:*")
}

func mapAuctionDetailRow(row repository.GetAuctionByIDRow) *model.Auction {
	images := parseImages(row.Images)
	return &model.Auction{
		ID:            util.UUIDToString(row.ID),
		SellerID:      util.UUIDToString(row.SellerID),
		SellerName:    row.SellerName,
		Title:         row.Title,
		Description:   row.Description.String,
		Images:        images,
		ImageURL:      firstImage(images),
		StartingPrice: row.StartingPrice,
		CurrentPrice:  row.CurrentPrice,
		WinnerID:      nullBytesToStringPtr(row.WinnerID),
		WinnerName:    row.WinnerName,
		Status:        model.AuctionStatus(row.Status),
		StartTime:     row.StartTime,
		EndTime:       row.EndTime,
		CreatedAt:     row.CreatedAt,
		BidCount:      int(row.BidCount),
	}
}

func parseImages(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var images []string
	if err := json.Unmarshal(raw, &images); err != nil {
		return []string{}
	}
	return images
}

func firstImage(images []string) string {
	if len(images) > 0 {
		return images[0]
	}
	return ""
}

func nullBytesToStringPtr(n sql.NullString) *string {
	if !n.Valid || n.String == "" {
		return nil
	}
	s := util.UUIDToString([]byte(n.String))
	if s == "" {
		return nil
	}
	return &s
}
