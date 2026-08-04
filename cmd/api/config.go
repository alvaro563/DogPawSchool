package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env             string
	Port            int
	JWTSecret       string
	CORSOrigins     []string
	TLSKeyFile      string
	TLSCertFile     string
	TrustedProxies  []string
	ShutdownTimeout time.Duration
	DB              DBConfig
}

type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	PingTimeout     time.Duration
}

// DSN renders the PostgreSQL Data Source Name for opening the connection
// pool with database/sql and the pgx stdlib adapter.
func (dbConfig DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbConfig.User, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Name, dbConfig.SSLMode,
	)
}

// LoadConfig reads configuration from environment variables, falling back
// to safe development defaults if any variable is missing or empty.
//
// JWT_SECRET must be set in production (returns an error otherwise). In
// development a placeholder is used with a warning.
//
// DB_SSLMODE defaults to empty (which the Postgres driver interprets as
// "prefer"). Production deployments should set it to "require" or
// "verify-full".
func LoadConfig() (Config, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	env := getEnv("ENV", "development")
	if env == "production" && jwtSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required in production")
	}
	if jwtSecret == "" {
		jwtSecret = "dev-secret"
	}

	sslMode := getEnv("DB_SSLMODE", "")
	if env == "production" && sslMode == "" {
		sslMode = "require"
	}

	corsOrigins := parseCSV(os.Getenv("CORS_ORIGINS"))
	trustedProxies := parseCSV(os.Getenv("TRUSTED_PROXIES"))

	cfg := Config{
		Env:             env,
		Port:            getEnvInt("PORT", 8080),
		JWTSecret:       jwtSecret,
		CORSOrigins:     corsOrigins,
		TLSKeyFile:      os.Getenv("TLS_KEY_FILE"),
		TLSCertFile:     os.Getenv("TLS_CERT_FILE"),
		TrustedProxies:  trustedProxies,
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "dogpaw_user"),
			Password:        getEnv("DB_PASSWORD", "dogpaw_pass"),
			Name:            getEnv("DB_NAME", "dogpaw_db"),
			SSLMode:         sslMode,
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			PingTimeout:     getEnvDuration("DB_PING_TIMEOUT", 30*time.Second),
		},
	}
	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		if parsedInt, err := strconv.Atoi(value); err == nil {
			return parsedInt
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if parsedDuration, err := time.ParseDuration(value); err == nil {
			return parsedDuration
		}
	}
	return defaultValue
}

func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
