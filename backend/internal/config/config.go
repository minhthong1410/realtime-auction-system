package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server  ServerConfig
	DB      DBConfig
	Redis   RedisConfig
	JWT     JWTConfig
	TOTP    TOTPConfig
	Stripe  StripeConfig
	Storage StorageConfig
	Grafana GrafanaConfig
}

type GrafanaConfig struct {
	RemoteWriteURL string
	Username       string
	APIKey         string
}

type StorageConfig struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	PublicURL string // public URL prefix for serving files
}

type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DBConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	URL      string // redis://default:password@host:port (preferred, e.g. Railway)
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type TOTPConfig struct {
	AESKey string // 32-byte key for AES-256-GCM encryption of TOTP secrets
	Issuer string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		DB: DBConfig{
			DSN:             parseDatabaseDSN(),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: 5 * time.Minute,
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", ""),
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		TOTP: TOTPConfig{
			AESKey: requireEnvOrDev("TOTP_AES_KEY", "01234567890123456789012345678901"),
			Issuer: getEnv("TOTP_ISSUER", "AuctionSystem"),
		},
		Storage: StorageConfig{
			Endpoint:  getEnv("S3_ENDPOINT", ""),
			Bucket:    getEnv("S3_BUCKET", "auction"),
			AccessKey: getEnv("S3_ACCESS_KEY", ""),
			SecretKey: getEnv("S3_SECRET_KEY", ""),
			PublicURL: getEnv("S3_PUBLIC_URL", ""),
		},
		Stripe: StripeConfig{
			SecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
			WebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		},
		Grafana: GrafanaConfig{
			RemoteWriteURL: getEnv("GRAFANA_REMOTE_WRITE_URL", ""),
			Username:       getEnv("GRAFANA_USERNAME", ""),
			APIKey:         getEnv("GRAFANA_API_KEY", ""),
		},
		JWT: JWTConfig{
			Secret:          requireEnvOrDev("JWT_SECRET", "dev-only-secret-do-not-use-in-prod"),
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
	}
}

// parseDatabaseDSN converts DATABASE_URL (mysql://user:pass@host:port/db) to Go MySQL DSN,
// or falls back to DATABASE_DSN if set directly.
func parseDatabaseDSN() string {
	// Direct DSN takes priority
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		return dsn
	}

	// Parse URL format (e.g. Railway: mysql://root:pass@host:port/railway)
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		u, err := url.Parse(dbURL)
		if err == nil {
			password, _ := u.User.Password()
			host := u.Host
			dbName := strings.TrimPrefix(u.Path, "/")
			return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&loc=UTC",
				u.User.Username(), password, host, dbName)
		}
	}

	return "root:root@tcp(localhost:3306)/auction?parseTime=true&loc=UTC"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// requireEnvOrDev returns the env var value, or the dev fallback only in dev mode.
// In production (GIN_MODE=release), it panics if the env var is not set.
func requireEnvOrDev(key, devFallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if os.Getenv("GIN_MODE") == "release" {
		panic(fmt.Sprintf("FATAL: %s environment variable is required in production", key))
	}
	return devFallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
