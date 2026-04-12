package worker

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kurama/auction-system/backend/internal/logger"
	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/util"
	"github.com/kurama/auction-system/backend/internal/ws"
	"go.uber.org/zap"
)

type AuctionCloser struct {
	queries *repository.Queries
	db      *sql.DB
	hub     *ws.Hub
}

func NewAuctionCloser(db *sql.DB, queries *repository.Queries, hub *ws.Hub) *AuctionCloser {
	return &AuctionCloser{queries: queries, db: db, hub: hub}
}

func (w *AuctionCloser) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	logger.Info("auction closer worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("auction closer worker stopped")
			return
		case <-ticker.C:
			if err := w.closeExpired(ctx); err != nil {
				logger.Error("auction closer error", zap.Error(err))
			}
		}
	}
}

func (w *AuctionCloser) closeExpired(ctx context.Context) error {
	expired, err := w.queries.GetExpiredActiveAuctions(ctx)
	if err != nil {
		return fmt.Errorf("get expired: %w", err)
	}

	if len(expired) == 0 {
		return nil
	}

	for _, auction := range expired {
		if err := w.closeOne(ctx, auction); err != nil {
			logger.Error("failed to close auction",
				zap.String("auction_id", util.UUIDToString(auction.ID)),
				zap.Error(err))
		}
	}

	return nil
}

func (w *AuctionCloser) closeOne(ctx context.Context, auction repository.GetExpiredActiveAuctionsRow) error {
	auctionIDStr := util.UUIDToString(auction.ID)

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := w.queries.WithTx(tx)

	// Close auction
	if err := qtx.CloseAuctionByID(ctx, auction.ID); err != nil {
		return fmt.Errorf("close auction: %w", err)
	}

	// Transfer winning bid amount to seller
	if auction.WinnerID.Valid && auction.CurrentPrice > 0 {
		err := qtx.RefundBalance(ctx, repository.RefundBalanceParams{
			Balance: auction.CurrentPrice,
			ID:      auction.SellerID,
		})
		if err != nil {
			return fmt.Errorf("transfer to seller: %w", err)
		}
		logger.Info("transferred to seller",
			zap.String("auction_id", auctionIDStr),
			zap.String("seller_id", util.UUIDToString(auction.SellerID)),
			zap.Int64("amount", auction.CurrentPrice))
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	// Broadcast after commit
	winnerName := ""
	if auction.WinnerID.Valid {
		winnerBytes := []byte(auction.WinnerID.String)
		user, err := w.queries.GetUserByID(ctx, winnerBytes)
		if err == nil {
			winnerName = user.Username
		}
	}

	w.hub.BroadcastToRoom(ctx, fmt.Sprintf("auction:%s", auctionIDStr), model.WSMessage{
		Type: "auction_ended",
		Data: model.WSAuctionEnded{
			AuctionID:  auctionIDStr,
			Winner:     winnerName,
			FinalPrice: auction.CurrentPrice,
		},
	})

	// Notify seller balance update
	sellerIDStr := util.UUIDToString(auction.SellerID)
	newBalance, _ := w.queries.GetBalance(ctx, auction.SellerID)
	w.hub.BroadcastToRoom(ctx, fmt.Sprintf("user:%s", sellerIDStr), model.WSMessage{
		Type: "balance_update",
		Data: model.WSBalanceUpdate{Balance: newBalance, Reason: "auction_sold"},
	})

	logger.Info("auction closed",
		zap.String("auction_id", auctionIDStr),
		zap.String("winner", winnerName),
		zap.Int64("final_price", auction.CurrentPrice))

	return nil
}
