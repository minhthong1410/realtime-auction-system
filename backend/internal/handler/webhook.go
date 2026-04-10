package handler

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kurama/auction-system/backend/internal/app"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/service"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
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
		slog.Error("failed to read webhook body", "error", err)
		c.Status(http.StatusBadRequest)
		return
	}

	event, err := webhook.ConstructEvent(body, c.GetHeader("Stripe-Signature"), h.webhookSecret)
	if err != nil {
		slog.Error("webhook signature verification failed", "error", err)
		c.Status(http.StatusBadRequest)
		return
	}

	ctx := c.Request.Context()

	switch event.Type {
	case stripe.EventTypePaymentIntentSucceeded:
		var pi stripe.PaymentIntent
		if err := pi.UnmarshalJSON(event.Data.Raw); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		if err := h.depositService.HandlePaymentSuccess(ctx, pi.ID); err != nil {
			slog.Error("handle payment success failed", "error", err, "payment_id", pi.ID)
			c.Status(http.StatusInternalServerError)
			return
		}

	case stripe.EventTypePaymentIntentPaymentFailed:
		var pi stripe.PaymentIntent
		if err := pi.UnmarshalJSON(event.Data.Raw); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		if err := h.depositService.HandlePaymentFailed(ctx, pi.ID); err != nil {
			slog.Error("handle payment failed", "error", err, "payment_id", pi.ID)
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	c.Status(http.StatusOK)
}
