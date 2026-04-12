package service

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"

	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/logger"
	"github.com/kurama/auction-system/backend/internal/metrics"
	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/util"
	"github.com/kurama/auction-system/backend/internal/ws"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"go.uber.org/zap"
)

type DepositService struct {
	queries     *repository.Queries
	db          *sql.DB
	hub         *ws.Hub
	frontendURL string
}

func NewDepositService(db *sql.DB, queries *repository.Queries, hub *ws.Hub, frontendURL string) *DepositService {
	return &DepositService{queries: queries, db: db, hub: hub, frontendURL: frontendURL}
}

func (s *DepositService) CreateDeposit(ctx context.Context, userID string, amount int64) (*model.DepositResponse, error) {
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(string(stripe.CurrencyUSD)),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Wallet Deposit"),
					},
					UnitAmount: stripe.Int64(amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.frontendURL + "/wallet?status=success"),
		CancelURL:  stripe.String(s.frontendURL + "/wallet?status=cancelled"),
	}

	sess, err := session.New(params)
	if err != nil {
		logger.Error("failed to create stripe checkout session", zap.Error(err))
		return nil, appErr.ErrorInternalServer
	}

	userIDBytes, _ := util.UUIDFromString(userID)
	depositID := util.NewUUID()
	err = s.queries.CreateDeposit(ctx, repository.CreateDepositParams{
		ID:              depositID,
		UserID:          userIDBytes,
		Amount:          amount,
		StripePaymentID: sess.ID,
	})
	if err != nil {
		logger.Error("failed to create deposit record", zap.Error(err))
		return nil, appErr.ErrorDatabase
	}

	return &model.DepositResponse{
		CheckoutURL:     sess.URL,
		StripePaymentID: sess.ID,
		Amount:          amount,
	}, nil
}

func (s *DepositService) HandlePaymentSuccess(ctx context.Context, stripePaymentID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return appErr.ErrorDatabase
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	// Lock deposit row to prevent double processing from webhook retries
	deposit, err := qtx.LockDepositByStripeID(ctx, stripePaymentID)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return appErr.ErrorDatabase
	}

	if deposit.Status != "pending" {
		return nil
	}

	err = qtx.UpdateDepositStatus(ctx, repository.UpdateDepositStatusParams{
		Status: "completed",
		ID:     deposit.ID,
	})
	if err != nil {
		return appErr.ErrorDatabase
	}

	_, err = qtx.DepositBalance(ctx, repository.DepositBalanceParams{
		Balance: deposit.Amount,
		ID:      deposit.UserID,
	})
	if err != nil {
		return appErr.ErrorDatabase
	}

	if err := tx.Commit(); err != nil {
		return appErr.ErrorDatabase
	}

	metrics.DepositsTotal.WithLabelValues("completed").Inc()
	userIDStr := util.UUIDToString(deposit.UserID)
	logger.Info("deposit completed via webhook", zap.String("deposit_id", util.UUIDToString(deposit.ID)), zap.String("user_id", userIDStr), zap.Int64("amount", deposit.Amount))

	newBalance, _ := s.queries.GetBalance(ctx, deposit.UserID)
	s.hub.BroadcastToRoom(ctx, fmt.Sprintf("user:%s", userIDStr), model.WSMessage{
		Type: "balance_update",
		Data: model.WSBalanceUpdate{Balance: newBalance, Reason: "deposit"},
	})

	return nil
}

func (s *DepositService) HandlePaymentFailed(ctx context.Context, stripePaymentID string) error {
	deposit, err := s.queries.GetDepositByStripeID(ctx, stripePaymentID)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return appErr.ErrorDatabase
	}

	if deposit.Status != "pending" {
		return nil
	}

	return s.queries.UpdateDepositStatus(ctx, repository.UpdateDepositStatusParams{
		Status: "failed",
		ID:     deposit.ID,
	})
}

func (s *DepositService) ListByUser(ctx context.Context, userID string, limit, offset int32) ([]model.Deposit, error) {
	userIDBytes, _ := util.UUIDFromString(userID)
	rows, err := s.queries.ListDepositsByUser(ctx, repository.ListDepositsByUserParams{
		UserID: userIDBytes,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, appErr.ErrorDatabase
	}

	deposits := make([]model.Deposit, len(rows))
	for i, row := range rows {
		deposits[i] = model.Deposit{
			ID:              util.UUIDToString(row.ID),
			UserID:          util.UUIDToString(row.UserID),
			Amount:          row.Amount,
			Status:          row.Status,
			StripePaymentID: row.StripePaymentID,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		}
	}

	return deposits, nil
}
