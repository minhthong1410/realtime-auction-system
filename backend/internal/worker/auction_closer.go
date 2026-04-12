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

	// Close in batch transaction
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := w.queries.WithTx(tx)

	for _, auction := range expired {
		if err := qtx.CloseAuctionByID(ctx, auction.ID); err != nil {
			logger.Error("failed to close auction", zap.String("auction_id", util.UUIDToString(auction.ID)), zap.Error(err))
			continue
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	// Broadcast after commit
	for _, auction := range expired {
		auctionIDStr := util.UUIDToString(auction.ID)
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

		logger.Info("auction closed", zap.String("auction_id", auctionIDStr), zap.String("winner", winnerName), zap.Int64("final_price", auction.CurrentPrice))
	}

	return nil
}
