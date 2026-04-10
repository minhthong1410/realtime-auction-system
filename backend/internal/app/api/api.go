package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v82"
	"go.uber.org/zap"

	"github.com/kurama/auction-system/backend/internal/app"
	"github.com/kurama/auction-system/backend/internal/config"
	"github.com/kurama/auction-system/backend/internal/database"
	"github.com/kurama/auction-system/backend/internal/handler"
	"github.com/kurama/auction-system/backend/internal/i18n"
	"github.com/kurama/auction-system/backend/internal/repository"
	"github.com/kurama/auction-system/backend/internal/middleware"
	"github.com/kurama/auction-system/backend/internal/storage"
	"github.com/kurama/auction-system/backend/internal/worker"
	"github.com/kurama/auction-system/backend/internal/ws"
)

type Application struct {
	cfg    *config.Config
	db     *sql.DB
	rdb    *redis.Client
	hub    *ws.Hub
	logger *zap.Logger
	engine *gin.Engine
}

func New(cfg *config.Config, logger *zap.Logger) *Application {
	return &Application{cfg: cfg, logger: logger}
}

func (a *Application) Run() error {
	i18n.Init("internal/i18n/locales")
	stripe.Key = a.cfg.Stripe.SecretKey

	if err := a.initDB(); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer a.db.Close()

	if err := database.RunMigrations(a.db, "db/migrations"); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	if err := a.initRedis(); err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	defer a.rdb.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// WebSocket Hub
	a.hub = ws.NewHub(a.rdb)
	go a.hub.Run(ctx)

	// Gin engine with global middleware
	a.engine = a.newRouter()

	// Build app context for handlers
	appCtx := a.buildContext()

	// Register all handlers (each registers its own routes)
	a.setupRoutes(appCtx)

	// Background workers
	queries := repository.New(a.db)
	auctionCloser := worker.NewAuctionCloser(a.db, queries, a.hub)
	go auctionCloser.Run(ctx)

	// HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", a.cfg.Server.Port),
		Handler:      a.engine,
		ReadTimeout:  a.cfg.Server.ReadTimeout,
		WriteTimeout: a.cfg.Server.WriteTimeout,
	}

	go func() {
		a.logger.Info("server starting", zap.String("port", a.cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.logger.Info("shutting down server...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	a.logger.Info("server stopped")
	return nil
}

func (a *Application) newRouter() *gin.Engine {
	r := gin.New()

	r.Use(middleware.RequestID())
	r.Use(middleware.ZapLogger(a.logger))
	r.Use(middleware.ZapRecovery(a.logger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.MaxBodySize(10 << 20))
	r.Use(middleware.I18nMiddleware())
	r.Use(middleware.RateLimit(100, time.Minute))

	allowedOrigins := []string{"http://localhost:3000", "http://localhost:5173"}
	if frontendURL := os.Getenv("FRONTEND_URL"); frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	r.GET("/health", a.healthCheck)

	return r
}

func (a *Application) buildContext() *app.Context {
	return &app.Context{
		Cfg:       a.cfg,
		DB:        a.db,
		Engine:    a.engine,
		Logger:    a.logger,
		Redis:     a.rdb,
		Hub:       a.hub,
		S3:        storage.NewS3Client(a.cfg.Storage),
		JWTSecret: a.cfg.JWT.Secret,
		Wrap:      middleware.WrapHandler,
		Auth:      middleware.AuthMiddleware(a.cfg.JWT.Secret),
	}
}

func (a *Application) setupRoutes(ctx *app.Context) {
	handler.NewAuthHandler(ctx)
	handler.NewAuctionHandler(ctx)
	handler.NewDepositHandler(ctx)
	handler.NewUploadHandler(ctx)
	handler.NewWebhookHandler(ctx)
	handler.NewWSHandler(ctx)
}

func (a *Application) healthCheck(c *gin.Context) {
	status := "ok"
	code := 200

	if err := a.db.Ping(); err != nil {
		status = "db_error"
		code = 503
	}

	if err := a.rdb.Ping(c.Request.Context()).Err(); err != nil {
		status = "redis_error"
		code = 503
	}

	c.JSON(code, gin.H{"status": status})
}

func (a *Application) initDB() error {
	db, err := sql.Open("mysql", a.cfg.DB.DSN)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(a.cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(a.cfg.DB.MaxIdleConns)
	db.SetConnMaxLifetime(a.cfg.DB.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return err
	}

	a.logger.Info("database connected")
	a.db = db
	return nil
}

func (a *Application) initRedis() error {
	var rdb *redis.Client

	if a.cfg.Redis.URL != "" {
		// Parse URL (e.g. redis://default:password@host:port)
		opt, err := redis.ParseURL(a.cfg.Redis.URL)
		if err != nil {
			return fmt.Errorf("parse redis url: %w", err)
		}
		rdb = redis.NewClient(opt)
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:     a.cfg.Redis.Addr,
			Password: a.cfg.Redis.Password,
			DB:       a.cfg.Redis.DB,
		})
	}

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return err
	}

	a.logger.Info("redis connected")
	a.rdb = rdb
	return nil
}
