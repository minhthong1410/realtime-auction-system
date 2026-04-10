package app

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/kurama/auction-system/backend/internal/config"
	"github.com/kurama/auction-system/backend/internal/middleware"
	"github.com/kurama/auction-system/backend/internal/storage"
	"github.com/kurama/auction-system/backend/internal/ws"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Context struct {
	Cfg       *config.Config
	DB        *sql.DB
	Engine    *gin.Engine
	Logger    *zap.Logger
	Redis     *redis.Client
	Hub       *ws.Hub
	S3        *storage.S3Client
	JWTSecret string
	Wrap      func(middleware.HandlerFunc) gin.HandlerFunc
	Auth      gin.HandlerFunc
}
