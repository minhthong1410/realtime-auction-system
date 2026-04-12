package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kurama/auction-system/backend/internal/app"
	"github.com/kurama/auction-system/backend/internal/logger"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/service"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
	"go.uber.org/zap"
)

type WebhookHandler struct {
	depositService *service.DepositService
	webhookSecret  string
}

func NewWebhookHandler(ctx *app.Context) *WebhookHandler {
	queries := repository.New(ctx.DB)
	frontendURL := getEnvOrDefault("FRONTEND_URL", "http://localhost:3000")
	depositService := service.NewDepositService(ctx.DB, queries, ctx.Hub, frontendURL)

	h := &WebhookHandler{
		depositService: depositService,
		webhookSecret:  ctx.Cfg.Stripe.WebhookSecret,
	}

	ctx.Engine.POST("/webhook/stripe", h.HandleStripeWebhook)

	return h
}

func (h *WebhookHandler) HandleStripeWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("failed to read webhook body", zap.Error(err))
		c.Status(http.StatusBadRequest)
		return
	}

	event, err := webhook.ConstructEventWithOptions(body, c.GetHeader("Stripe-Signature"), h.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		logger.Error("webhook signature verification failed", zap.Error(err))
		c.Status(http.StatusBadRequest)
		return
	}

	ctx := c.Request.Context()

	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		var sess stripe.CheckoutSession
		if err := sess.UnmarshalJSON(event.Data.Raw); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		if sess.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
			if err := h.depositService.HandlePaymentSuccess(ctx, sess.ID); err != nil {
				logger.Error("handle payment success failed", zap.Error(err), zap.String("session_id", sess.ID))
				c.Status(http.StatusInternalServerError)
				return
			}
		}

	case stripe.EventTypeCheckoutSessionExpired:
		var sess stripe.CheckoutSession
		if err := sess.UnmarshalJSON(event.Data.Raw); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		if err := h.depositService.HandlePaymentFailed(ctx, sess.ID); err != nil {
			logger.Error("handle payment failed", zap.Error(err), zap.String("session_id", sess.ID))
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	c.Status(http.StatusOK)
}
