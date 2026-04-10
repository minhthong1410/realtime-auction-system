package i18n

import (
	"embed"
	"encoding/json"
	"log/slog"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
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
		slog.Error("failed to read embedded i18n directory", "error", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := localesFS.ReadFile("locales/" + entry.Name())
		if err != nil {
			slog.Error("failed to read i18n file", "file", entry.Name(), "error", err)
			continue
		}
		if _, err := Bundle.ParseMessageFileBytes(data, entry.Name()); err != nil {
			slog.Error("failed to parse i18n file", "file", entry.Name(), "error", err)
		} else {
			slog.Info("loaded i18n file", "file", entry.Name())
		}
	}
}
