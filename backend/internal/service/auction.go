package service

import (
	"context"
	"database/sql"
	stderrors "errors"
	"time"

	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/util"
)

type AuctionService struct {
	queries *repository.Queries
	db      *sql.DB
}

func NewAuctionService(db *sql.DB, queries *repository.Queries) *AuctionService {
	return &AuctionService{queries: queries, db: db}
}

func (s *AuctionService) Create(ctx context.Context, userID string, req model.CreateAuctionRequest) (*model.Auction, error) {
	if req.EndTime.Before(time.Now().Add(time.Minute)) {
		return nil, appErr.ErrorInvalidEndTime
	}

	userIDBytes, _ := util.UUIDFromString(userID)
	auctionID := util.NewUUID()

	err := s.queries.CreateAuction(ctx, repository.CreateAuctionParams{
		ID:            auctionID,
		SellerID:      userIDBytes,
		Title:         req.Title,
		Description:   sql.NullString{String: req.Description, Valid: req.Description != ""},
		ImageUrl:      sql.NullString{String: req.ImageURL, Valid: req.ImageURL != ""},
		StartingPrice: req.StartingPrice,
		CurrentPrice:  req.StartingPrice,
		EndTime:       req.EndTime,
	})
	if err != nil {
		return nil, appErr.ErrorDatabase
	}

	return s.GetByID(ctx, util.UUIDToString(auctionID))
}

func (s *AuctionService) GetByID(ctx context.Context, id string) (*model.Auction, error) {
	idBytes, err := util.UUIDFromString(id)
	if err != nil {
		return nil, appErr.ErrorAuctionNotFound
	}

	row, err := s.queries.GetAuctionByID(ctx, idBytes)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, appErr.ErrorAuctionNotFound
		}
		return nil, appErr.ErrorDatabase
	}

	return mapAuctionDetailRow(row), nil
}

func (s *AuctionService) ListActive(ctx context.Context, limit, offset int32) ([]model.Auction, int64, error) {
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
		auctions[i] = model.Auction{
			ID:            util.UUIDToString(row.ID),
			SellerID:      util.UUIDToString(row.SellerID),
			SellerName:    row.SellerName,
			Title:         row.Title,
			Description:   row.Description.String,
			ImageURL:      row.ImageUrl.String,
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
		auctions[i] = model.Auction{
			ID:            util.UUIDToString(row.ID),
			SellerID:      util.UUIDToString(row.SellerID),
			SellerName:    row.SellerName,
			Title:         row.Title,
			Description:   row.Description.String,
			ImageURL:      row.ImageUrl.String,
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

func mapAuctionDetailRow(row repository.GetAuctionByIDRow) *model.Auction {
	return &model.Auction{
		ID:            util.UUIDToString(row.ID),
		SellerID:      util.UUIDToString(row.SellerID),
		SellerName:    row.SellerName,
		Title:         row.Title,
		Description:   row.Description.String,
		ImageURL:      row.ImageUrl.String,
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

// nullBytesToStringPtr converts sql.NullString (MySQL scans BINARY nullable as NullString) to *string UUID.
func nullBytesToStringPtr(n sql.NullString) *string {
	if !n.Valid || n.String == "" {
		return nil
	}
	// MySQL driver may return raw bytes as string for BINARY columns
	s := util.UUIDToString([]byte(n.String))
	if s == "" {
		return nil
	}
	return &s
}
