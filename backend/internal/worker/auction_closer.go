package worker

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/util"
	"github.com/kurama/auction-system/backend/internal/ws"
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

	slog.Info("auction closer worker started")

	for {
		select {
		case <-ctx.Done():
			slog.Info("auction closer worker stopped")
			return
		case <-ticker.C:
			if err := w.closeExpired(ctx); err != nil {
				slog.Error("auction closer error", "error", err)
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
			slog.Error("failed to close auction", "auction_id", util.UUIDToString(auction.ID), "error", err)
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

		slog.Info("auction closed", "auction_id", auctionIDStr, "winner", winnerName, "final_price", auction.CurrentPrice)
	}

	return nil
}
