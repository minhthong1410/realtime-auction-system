package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/kurama/auction-system/backend/internal/app/api"
	"github.com/kurama/auction-system/backend/internal/config"
	"go.uber.org/zap"
)

func main() {
	// Load .env file (ignore error if not found — production uses real env vars)
	_ = godotenv.Load()

	var logger *zap.Logger
	var err error

	if os.Getenv("GIN_MODE") == "release" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}
	defer logger.Sync()

	cfg := config.Load()

	app := api.New(cfg, logger)
	if err := app.Run(); err != nil {
		logger.Fatal("application error", zap.Error(err))
	}
}
