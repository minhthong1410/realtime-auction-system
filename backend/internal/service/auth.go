package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kurama/auction-system/backend/internal/config"
	appErr "github.com/kurama/auction-system/backend/internal/errors"
	"github.com/kurama/auction-system/backend/internal/model"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/util"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	queries *repository.Queries
	db      *sql.DB
	cfg     config.JWTConfig
}

func NewAuthService(db *sql.DB, queries *repository.Queries, cfg config.JWTConfig) *AuthService {
	return &AuthService{queries: queries, db: db, cfg: cfg}
}

func (s *AuthService) Register(ctx context.Context, req model.RegisterRequest) (*model.User, error) {
	_, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err == nil {
		return nil, appErr.ErrorUserAlreadyExists
	}
	_, err = s.queries.GetUserByUsername(ctx, req.Username)
	if err == nil {
		return nil, appErr.ErrorUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, appErr.ErrorInternalServer
	}

	id := util.NewUUID()
	err = s.queries.CreateUser(ctx, repository.CreateUserParams{
		ID:       id,
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
	})
	if err != nil {
		return nil, appErr.ErrorDatabase
	}

	return &model.User{
		ID:       util.UUIDToString(id),
		Username: req.Username,
		Email:    req.Email,
		Balance:  0,
	}, nil
}

// dummyHash is a bcrypt hash used to prevent timing-based user enumeration.
// When a user is not found, we still run bcrypt.CompareHashAndPassword against this
// dummy hash so the response time is consistent whether or not the email exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

// ValidateCredentials checks email/password and returns user (without issuing tokens).
func (s *AuthService) ValidateCredentials(ctx context.Context, req model.LoginRequest) (*model.User, error) {
	dbUser, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// Constant-time: run bcrypt even if user not found to prevent timing attack
		bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		return nil, appErr.ErrorInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(req.Password)); err != nil {
		return nil, appErr.ErrorInvalidCredentials
	}

	u := userFromRow(dbUser.ID, dbUser.Username, dbUser.Email, dbUser.Balance, dbUser.CreatedAt)
	return &u, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*model.TokenResponse, error) {
	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, appErr.ErrorInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, appErr.ErrorInvalidToken
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return nil, appErr.ErrorInvalidToken
	}

	userID, _ := claims["user_id"].(string)
	idBytes, err := util.UUIDFromString(userID)
	if err != nil {
		return nil, appErr.ErrorInvalidToken
	}

	dbUser, err := s.queries.GetUserByID(ctx, idBytes)
	if err != nil {
		return nil, appErr.ErrorInvalidToken
	}

	return s.generateTokens(userFromRow(dbUser.ID, dbUser.Username, dbUser.Email, dbUser.Balance, dbUser.CreatedAt))
}

func (s *AuthService) GetProfile(ctx context.Context, userID string) (*model.User, error) {
	idBytes, err := util.UUIDFromString(userID)
	if err != nil {
		return nil, appErr.ErrorNotFound
	}
	dbUser, err := s.queries.GetUserByID(ctx, idBytes)
	if err != nil {
		return nil, appErr.ErrorNotFound
	}
	u := userFromRow(dbUser.ID, dbUser.Username, dbUser.Email, dbUser.Balance, dbUser.CreatedAt)
	return &u, nil
}

// GenerateTokensForUser is a public wrapper for generating JWT tokens.
func (s *AuthService) GenerateTokensForUser(user model.User) (*model.TokenResponse, error) {
	return s.generateTokens(user)
}

func (s *AuthService) generateTokens(user model.User) (*model.TokenResponse, error) {
	now := time.Now()

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"type":     "access",
		"exp":      now.Add(s.cfg.AccessTokenTTL).Unix(),
		"iat":      now.Unix(),
	})
	accessStr, err := accessToken.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, appErr.ErrorInternalServer
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"type":    "refresh",
		"exp":     now.Add(s.cfg.RefreshTokenTTL).Unix(),
		"iat":     now.Unix(),
	})
	refreshStr, err := refreshToken.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, appErr.ErrorInternalServer
	}

	return &model.TokenResponse{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		User:         user,
	}, nil
}

func userFromRow(id []byte, username, email string, balance int64, createdAt time.Time) model.User {
	return model.User{
		ID:        util.UUIDToString(id),
		Username:  username,
		Email:     email,
		Balance:   balance,
		CreatedAt: createdAt,
	}
}
