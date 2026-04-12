package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	assert.Equal(t, "fallback", getEnv("NONEXISTENT_KEY_12345", "fallback"))

	os.Setenv("TEST_CONFIG_KEY", "value")
	defer os.Unsetenv("TEST_CONFIG_KEY")
	assert.Equal(t, "value", getEnv("TEST_CONFIG_KEY", "fallback"))
}

func TestGetEnvInt(t *testing.T) {
	assert.Equal(t, 42, getEnvInt("NONEXISTENT_INT_KEY", 42))

	os.Setenv("TEST_INT_KEY", "100")
	defer os.Unsetenv("TEST_INT_KEY")
	assert.Equal(t, 100, getEnvInt("TEST_INT_KEY", 42))

	os.Setenv("TEST_INT_INVALID", "not_a_number")
	defer os.Unsetenv("TEST_INT_INVALID")
	assert.Equal(t, 42, getEnvInt("TEST_INT_INVALID", 42))
}

func TestRequireEnvOrDevInDevMode(t *testing.T) {
	os.Unsetenv("GIN_MODE")
	defer os.Unsetenv("GIN_MODE")

	// Dev mode: should return fallback
	result := requireEnvOrDev("NONEXISTENT_SECRET", "dev-fallback")
	assert.Equal(t, "dev-fallback", result)
}

func TestRequireEnvOrDevWithEnvSet(t *testing.T) {
	os.Setenv("TEST_REQUIRED_KEY", "real-value")
	defer os.Unsetenv("TEST_REQUIRED_KEY")

	result := requireEnvOrDev("TEST_REQUIRED_KEY", "fallback")
	assert.Equal(t, "real-value", result)
}

func TestRequireEnvOrDevPanicsInProduction(t *testing.T) {
	os.Setenv("GIN_MODE", "release")
	defer os.Unsetenv("GIN_MODE")
	os.Unsetenv("NONEXISTENT_PROD_KEY")

	assert.Panics(t, func() {
		requireEnvOrDev("NONEXISTENT_PROD_KEY", "fallback")
	})
}

func TestRequireEnvOrDevDoesNotPanicWhenSet(t *testing.T) {
	os.Setenv("GIN_MODE", "release")
	defer os.Unsetenv("GIN_MODE")
	os.Setenv("PROD_KEY_SET", "secret")
	defer os.Unsetenv("PROD_KEY_SET")

	assert.NotPanics(t, func() {
		result := requireEnvOrDev("PROD_KEY_SET", "fallback")
		assert.Equal(t, "secret", result)
	})
}

func TestParseDatabaseDSN(t *testing.T) {
	// Default (no env vars)
	os.Unsetenv("DATABASE_DSN")
	os.Unsetenv("DATABASE_URL")
	dsn := parseDatabaseDSN()
	assert.Contains(t, dsn, "root:root@tcp(localhost:3306)/auction")

	// Direct DSN
	os.Setenv("DATABASE_DSN", "custom:dsn@tcp(host:3306)/db")
	defer os.Unsetenv("DATABASE_DSN")
	assert.Equal(t, "custom:dsn@tcp(host:3306)/db", parseDatabaseDSN())

	// URL format (Railway style)
	os.Unsetenv("DATABASE_DSN")
	os.Setenv("DATABASE_URL", "mysql://root:pass@host:3306/railway")
	defer os.Unsetenv("DATABASE_URL")
	dsn = parseDatabaseDSN()
	assert.Equal(t, "root:pass@tcp(host:3306)/railway?parseTime=true&loc=UTC", dsn)
}

func TestLoadReturnsConfig(t *testing.T) {
	os.Unsetenv("GIN_MODE") // ensure dev mode
	cfg := Load()

	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, 25, cfg.DB.MaxOpenConns)
	assert.Equal(t, "AuctionSystem", cfg.TOTP.Issuer)
	assert.NotEmpty(t, cfg.JWT.Secret)
	assert.NotEmpty(t, cfg.TOTP.AESKey)
}
