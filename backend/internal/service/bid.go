package service

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"
	"log/slog"
	"time"

	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/util"
	"github.com/kurama/auction-system/backend/internal/ws"
)

type BidService struct {
	queries *repository.Queries
	db      *sql.DB
	hub     *ws.Hub
}

func NewBidService(db *sql.DB, queries *repository.Queries, hub *ws.Hub) *BidService {
	return &BidService{queries: queries, db: db, hub: hub}
}

func (s *BidService) PlaceBid(ctx context.Context, auctionID, userID string, username string, amount int64) (*model.Bid, error) {
	auctionIDBytes, _ := util.UUIDFromString(auctionID)
	userIDBytes, _ := util.UUIDFromString(userID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, appErr.ErrorDatabase
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	auction, err := qtx.LockAuctionForBid(ctx, auctionIDBytes)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, appErr.ErrorAuctionNotFound
		}
		return nil, appErr.ErrorDatabase
	}

	sellerID := util.UUIDToString(auction.SellerID)
	if sellerID == userID {
		return nil, appErr.ErrorSelfBid
	}
	if auction.EndTime.Before(time.Now()) {
		return nil, appErr.ErrorAuctionEnded
	}
	if amount <= auction.CurrentPrice {
		return nil, appErr.ErrorBidTooLow
	}

	deductResult, err := qtx.DeductBalance(ctx, repository.DeductBalanceParams{
		Balance:   amount,
		ID:        userIDBytes,
		Balance_2: amount,
	})
	if err != nil {
		return nil, appErr.ErrorDatabase
	}
	rowsAffected, _ := deductResult.RowsAffected()
	if rowsAffected == 0 {
		return nil, appErr.ErrorInsufficientFunds
	}

	// Refund previous highest bidder
	prevWinnerID := ""
	if auction.WinnerID.Valid {
		prevWinnerBytes := []byte(auction.WinnerID.String)
		prevWinnerID = util.UUIDToString(prevWinnerBytes)
		err = qtx.RefundBalance(ctx, repository.RefundBalanceParams{
			Balance: auction.CurrentPrice,
			ID:      prevWinnerBytes,
		})
		if err != nil {
			return nil, appErr.ErrorDatabase
		}
	}

	err = qtx.UpdateAuctionBid(ctx, repository.UpdateAuctionBidParams{
		ID:           auctionIDBytes,
		CurrentPrice: amount,
		WinnerID:     sql.NullString{String: string(userIDBytes), Valid: true},
	})
	if err != nil {
		return nil, appErr.ErrorDatabase
	}

	bidID := util.NewUUID()
	err = qtx.CreateBid(ctx, repository.CreateBidParams{
		ID:        bidID,
		AuctionID: auctionIDBytes,
		UserID:    userIDBytes,
		Amount:    amount,
	})
	if err != nil {
		return nil, appErr.ErrorDatabase
	}

	if err := tx.Commit(); err != nil {
		return nil, appErr.ErrorDatabase
	}

	bidCount, _ := s.queries.CountBidsByAuction(ctx, auctionIDBytes)

	bid := &model.Bid{
		ID:        util.UUIDToString(bidID),
		AuctionID: auctionID,
		UserID:    userID,
		Username:  username,
		Amount:    amount,
		CreatedAt: time.Now(),
	}

	// Broadcast new bid
	timeLeft := time.Until(auction.EndTime).Seconds()
	s.hub.BroadcastToRoom(ctx, fmt.Sprintf("auction:%s", auctionID), model.WSMessage{
		Type: "new_bid",
		Data: model.WSNewBid{
			AuctionID: auctionID,
			Amount:    amount,
			Username:  username,
			BidCount:  int(bidCount),
			TimeLeft:  int64(timeLeft),
			CreatedAt: bid.CreatedAt,
		},
	})

	// Broadcast balance to bidder
	newBalance, _ := s.queries.GetBalance(ctx, userIDBytes)
	s.hub.BroadcastToRoom(ctx, fmt.Sprintf("user:%s", userID), model.WSMessage{
		Type: "balance_update",
		Data: model.WSBalanceUpdate{Balance: newBalance, Reason: "bid_placed"},
	})

	// Broadcast refund to previous winner
	if prevWinnerID != "" {
		prevWinnerBytes, _ := util.UUIDFromString(prevWinnerID)
		refundedBalance, _ := s.queries.GetBalance(ctx, prevWinnerBytes)
		s.hub.BroadcastToRoom(ctx, fmt.Sprintf("user:%s", prevWinnerID), model.WSMessage{
			Type: "balance_update",
			Data: model.WSBalanceUpdate{Balance: refundedBalance, Reason: "bid_refund"},
		})
	}

	slog.Info("bid placed", "auction_id", auctionID, "user_id", userID, "amount", amount)
	return bid, nil
}
