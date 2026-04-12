package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/kurama/auction-system/backend/internal/app/api"
	"github.com/kurama/auction-system/backend/internal/config"
	"github.com/kurama/auction-system/backend/internal/logger"
	"go.uber.org/zap"
)

func main() {
	// Load .env file (ignore error if not found — production uses real env vars)
	_ = godotenv.Load()

	var l *zap.Logger
	var err error

	if os.Getenv("GIN_MODE") == "release" {
		l, err = zap.NewProduction()
	} else {
		l, err = zap.NewDevelopment()
	}
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}
	defer l.Sync()

	logger.Init(l)

	cfg := config.Load()

	app := api.New(cfg, l)
	if err := app.Run(); err != nil {
		l.Fatal("application error", zap.Error(err))
	}
}
