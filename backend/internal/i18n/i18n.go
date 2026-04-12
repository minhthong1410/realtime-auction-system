package i18n

import (
	"embed"
	"encoding/json"
	"path/filepath"

	"github.com/kurama/auction-system/backend/internal/logger"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/zap"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localesFS embed.FS

var Bundle *i18n.Bundle

func Init() {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	entries, err := localesFS.ReadDir("locales")
	if err != nil {
		logger.Error("failed to read embedded i18n directory", zap.Error(err))
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := localesFS.ReadFile("locales/" + entry.Name())
		if err != nil {
			logger.Error("failed to read i18n file", zap.String("file", entry.Name()), zap.Error(err))
			continue
		}
		if _, err := Bundle.ParseMessageFileBytes(data, entry.Name()); err != nil {
			logger.Error("failed to parse i18n file", zap.String("file", entry.Name()), zap.Error(err))
		} else {
			logger.Info("loaded i18n file", zap.String("file", entry.Name()))
		}
	}
}
