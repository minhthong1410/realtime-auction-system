package database

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/kurama/auction-system/backend/internal/logger"
	"go.uber.org/zap"
)

func RunMigrations(db *sql.DB, migrationsDir string) error {
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsDir),
		"mysql",
		driver,
	)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	// Fix dirty state if previous migration failed
	version, dirty, _ := m.Version()
	if dirty {
		logger.Info("fixing dirty migration", zap.Uint("version", version))
		m.Force(int(version) - 1)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	version, dirty, _ = m.Version()
	logger.Info("migrations completed", zap.Uint("version", version), zap.Bool("dirty", dirty))
	return nil
}
