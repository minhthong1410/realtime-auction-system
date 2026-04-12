package service

import (
	"context"
	"database/sql"
	"fmt"

	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/logger"
	"github.com/kurama/auction-system/backend/internal/metrics"
	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/util"
	"github.com/kurama/auction-system/backend/internal/ws"
	"go.uber.org/zap"
)

const minWithdrawalAmount int64 = 500 // $5.00 in cents

type WithdrawalService struct {
	queries *repository.Queries
	db      *sql.DB
	hub     *ws.Hub
}

func NewWithdrawalService(db *sql.DB, queries *repository.Queries, hub *ws.Hub) *WithdrawalService {
	return &WithdrawalService{queries: queries, db: db, hub: hub}
}

func (s *WithdrawalService) CreateWithdrawal(ctx context.Context, userID string, req model.WithdrawalRequest) (*model.Withdrawal, error) {
	if req.Amount < minWithdrawalAmount {
		return nil, appErr.ErrorWithdrawalMinAmount
	}

	userIDBytes, err := util.UUIDFromString(userID)
	if err != nil {
		return nil, appErr.ErrorInvalidParams
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, appErr.ErrorDatabase
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	// Lock user row to prevent concurrent withdrawal race condition
	_, err = qtx.LockUserForWithdrawal(ctx, userIDBytes)
	if err != nil {
		return nil, appErr.ErrorNotFound
	}

	// Check no pending withdrawal (now safe — user row is locked)
	pendingCount, err := qtx.CountPendingWithdrawalsByUser(ctx, userIDBytes)
	if err != nil {
		logger.Error("withdrawal: failed to check pending", zap.String("user_id", userID), zap.Error(err))
		return nil, appErr.ErrorDatabase
	}
	if pendingCount > 0 {
		return nil, appErr.ErrorWithdrawalPendingExists
	}

	// Deduct balance
	result, err := qtx.DeductBalance(ctx, repository.DeductBalanceParams{
		Balance:   req.Amount,
		ID:        userIDBytes,
		Balance_2: req.Amount,
	})
	if err != nil {
		logger.Error("withdrawal: failed to deduct balance", zap.String("user_id", userID), zap.Error(err))
		return nil, appErr.ErrorDatabase
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, appErr.ErrorInsufficientFunds
	}

	// Create withdrawal record
	withdrawalID := util.NewUUID()
	err = qtx.CreateWithdrawal(ctx, repository.CreateWithdrawalParams{
		ID:            withdrawalID,
		UserID:        userIDBytes,
		Amount:        req.Amount,
		BankName:      req.BankName,
		AccountNumber: req.AccountNumber,
		AccountHolder: req.AccountHolder,
		Note:          req.Note,
	})
	if err != nil {
		logger.Error("withdrawal: failed to create record", zap.String("user_id", userID), zap.Error(err))
		return nil, appErr.ErrorDatabase
	}

	if err := tx.Commit(); err != nil {
		logger.Error("withdrawal: failed to commit", zap.String("user_id", userID), zap.Error(err))
		return nil, appErr.ErrorDatabase
	}

	metrics.WithdrawalsTotal.WithLabelValues("pending").Inc()
	logger.Info("withdrawal created", zap.String("withdrawal_id", util.UUIDToString(withdrawalID)), zap.String("user_id", userID), zap.Int64("amount", req.Amount))

	// Broadcast balance update
	newBalance, _ := s.queries.GetBalance(ctx, userIDBytes)
	s.hub.BroadcastToRoom(ctx, fmt.Sprintf("user:%s", userID), model.WSMessage{
		Type: "balance_update",
		Data: model.WSBalanceUpdate{Balance: newBalance, Reason: "withdrawal"},
	})

	return &model.Withdrawal{
		ID:            util.UUIDToString(withdrawalID),
		UserID:        userID,
		Amount:        req.Amount,
		Status:        "pending",
		BankName:      req.BankName,
		AccountNumber: req.AccountNumber,
		AccountHolder: req.AccountHolder,
		Note:          req.Note,
	}, nil
}

func (s *WithdrawalService) ListByUser(ctx context.Context, userID string, limit, offset int32) ([]model.Withdrawal, error) {
	userIDBytes, _ := util.UUIDFromString(userID)
	rows, err := s.queries.ListWithdrawalsByUser(ctx, repository.ListWithdrawalsByUserParams{
		UserID: userIDBytes,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, appErr.ErrorDatabase
	}

	withdrawals := make([]model.Withdrawal, len(rows))
	for i, row := range rows {
		w := model.Withdrawal{
			ID:            util.UUIDToString(row.ID),
			UserID:        util.UUIDToString(row.UserID),
			Amount:        row.Amount,
			Status:        row.Status,
			BankName:      row.BankName,
			AccountNumber: maskAccountNumber(row.AccountNumber),
			AccountHolder: row.AccountHolder,
			Note:          row.Note,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		}
		if row.ReviewedAt.Valid {
			w.ReviewedAt = &row.ReviewedAt.Time
		}
		withdrawals[i] = w
	}

	return withdrawals, nil
}

// maskAccountNumber masks all but the last 4 characters: "123456789" → "****6789"
func maskAccountNumber(num string) string {
	if len(num) <= 4 {
		return num
	}
	masked := make([]byte, len(num))
	for i := range masked {
		masked[i] = '*'
	}
	copy(masked[len(num)-4:], num[len(num)-4:])
	return string(masked)
}
