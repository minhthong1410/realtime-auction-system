package i18n

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var Bundle *i18n.Bundle

func Init(localesDir string) {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	entries, err := os.ReadDir(localesDir)
	if err != nil {
		slog.Error("failed to read i18n directory", "dir", localesDir, "error", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(localesDir, entry.Name())
		if _, err := Bundle.LoadMessageFile(path); err != nil {
			slog.Error("failed to load i18n file", "file", path, "error", err)
		} else {
			slog.Info("loaded i18n file", "file", entry.Name())
		}
	}
}
