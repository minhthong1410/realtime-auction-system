package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kurama/auction-system/backend/internal/app"
	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/httputil"
	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
	totpService *service.TOTPService
}

func NewAuthHandler(ctx *app.Context) *AuthHandler {
	queries := repository.New(ctx.DB)
	authService := service.NewAuthService(ctx.DB, queries, ctx.Cfg.JWT)
	totpService := service.NewTOTPService(ctx.DB, queries, ctx.Redis, ctx.Cfg.TOTP, ctx.Cfg.JWT.Secret)

	h := &AuthHandler{
		authService: authService,
		totpService: totpService,
	}

	w := ctx.Wrap
	auth := ctx.Engine.Group("/api/auth")
	{
		auth.POST("/register", w(h.Register))
		auth.POST("/login", w(h.Login))
		auth.POST("/refresh", w(h.Refresh))
		auth.POST("/totp/setup", w(h.TotpSetup))
		auth.POST("/totp/confirm", w(h.TotpConfirm))
		auth.POST("/verify-otp", w(h.VerifyOTP))
	}

	protected := ctx.Engine.Group("/api")
	protected.Use(ctx.Auth)
	{
		protected.GET("/me", w(h.GetProfile))
		protected.POST("/totp/disable", w(h.TotpDisable))
	}

	return h
}

func (h *AuthHandler) Register(c *gin.Context) error {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	user, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		return renderServiceError(c, err)
	}

	// Register always requires TOTP setup before getting real JWT
	tempToken, err := h.totpService.GenerateTempToken(user.ID, user.Username, "totp_setup")
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusCreated, c, httputil.NewCreatedResponse(c, model.LoginResponse{
		RequireTotpSetup: true,
		TotpEnabled:      false,
		TempToken:        tempToken,
	}))
}

func (h *AuthHandler) Login(c *gin.Context) error {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	user, err := h.authService.ValidateCredentials(c.Request.Context(), req)
	if err != nil {
		return renderServiceError(c, err)
	}

	totpEnabled, _ := h.totpService.IsTOTPEnabled(c.Request.Context(), user.ID)

	purpose := "totp_setup"
	if totpEnabled {
		purpose = "totp_verify"
	}

	tempToken, err := h.totpService.GenerateTempToken(user.ID, user.Username, purpose)
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, model.LoginResponse{
		RequireTotpSetup: !totpEnabled,
		TotpEnabled:      totpEnabled,
		TempToken:        tempToken,
	}))
}

func (h *AuthHandler) Refresh(c *gin.Context) error {
	var req model.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	resp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, resp))
}

func (h *AuthHandler) GetProfile(c *gin.Context) error {
	userID := httputil.GetUserIDFromContext(c)
	user, err := h.authService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, user))
}

// --- TOTP ---

func (h *AuthHandler) TotpSetup(c *gin.Context) error {
	var req model.TotpSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	claims, err := h.totpService.ValidateTempToken(req.TempToken, "totp_setup")
	if err != nil {
		return renderServiceError(c, err)
	}

	qr, secret, err := h.totpService.SetupTOTP(c.Request.Context(), claims.UserID, claims.Username)
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, model.TotpSetupResponse{
		QRCode: qr,
		Secret: secret,
	}))
}

func (h *AuthHandler) TotpConfirm(c *gin.Context) error {
	var req model.TotpConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	claims, err := h.totpService.ValidateTempToken(req.TempToken, "totp_setup")
	if err != nil {
		return renderServiceError(c, err)
	}

	backupCodes, err := h.totpService.ConfirmTOTP(c.Request.Context(), claims.UserID, req.Code)
	if err != nil {
		return renderServiceError(c, err)
	}

	user, err := h.authService.GetProfile(c.Request.Context(), claims.UserID)
	if err != nil {
		return renderServiceError(c, err)
	}

	tokens, err := h.authService.GenerateTokensForUser(*user)
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, model.TotpConfirmResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         tokens.User,
		BackupCodes:  backupCodes,
	}))
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) error {
	var req model.VerifyOtpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return httputil.RenderError(c, appErr.ErrorInvalidParams)
	}

	claims, err := h.totpService.ValidateTempToken(req.TempToken, "totp_verify")
	if err != nil {
		return renderServiceError(c, err)
	}

	if err := h.totpService.VerifyTOTP(c.Request.Context(), claims.UserID, req.Code); err != nil {
		return renderServiceError(c, err)
	}

	user, err := h.authService.GetProfile(c.Request.Context(), claims.UserID)
	if err != nil {
		return renderServiceError(c, err)
	}

	tokens, err := h.authService.GenerateTokensForUser(*user)
	if err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, tokens))
}

func (h *AuthHandler) TotpDisable(c *gin.Context) error {
	userID := httputil.GetUserIDFromContext(c)
	if err := h.totpService.DisableTOTP(c.Request.Context(), userID); err != nil {
		return renderServiceError(c, err)
	}

	return httputil.RenderGinJSON(http.StatusOK, c, httputil.NewSuccessResponse(c, nil))
}
