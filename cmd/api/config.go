package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Secret length thresholds. JWT_SECRET matches the OWASP recommendation
// for HS256 (256 bits of entropy, i.e. 32 random bytes). DB_PASSWORD
// matches the OWASP password baseline (12 chars minimum).
const (
	minJWTSecretBytes  = 32
	minDBPasswordBytes = 12
)

// Closed sets of accepted values for env-typed fields. Anything else
// makes LoadConfig fail fast — typos like "producton" should NOT
// silently fall through to "development" semantics.
var (
	validEnvValues = map[string]struct{}{
		"development": {},
		"staging":     {},
		"production":  {},
	}
	validSSLModes = map[string]struct{}{
		"disable":     {},
		"allow":       {},
		"prefer":      {},
		"require":     {},
		"verify-ca":   {},
		"verify-full": {},
	}
)

type Config struct {
	Env             string
	Port            int
	JWTSecret       string
	JWTAccessTTL    time.Duration
	JWTRefreshTTL   time.Duration
	CookieSecure    bool
	CORSOrigins     []string
	TLSKeyFile      string
	TLSCertFile     string
	TrustedProxies  []string
	ShutdownTimeout time.Duration

	// IP rate limit on /auth/login. Buckets are in-memory, keyed by
	// TCP peer (RemoteAddr); see rateLimitMiddleware for the trust
	// model. Defaults are intentionally generous so legitimate
	// retries across flaky mobile networks aren't punished.
	LoginIPRateLimitPerMinute int
	LoginIPBurst              int

	// IP rate limit on /auth/register. Tighter than login because
	// each registration consumes an admin-issued invitation
	// token.
	RegisterIPRateLimitPerMinute int
	RegisterIPBurst              int

	// Account lockout (Postgres-backed, see AccountLoginLimiter).
	// Email and IP axes are evaluated independently; either one
	// can lock out a request.
	LockoutEmailMaxFailures int
	LockoutEmailWindow      time.Duration
	LockoutIPMaxFailures    int
	LockoutIPWindow         time.Duration

	// LoginAttemptsCleanupInterval drives the background
	// housekeeping goroutine that deletes login_attempts rows
	// older than LockoutEmailWindow*2 (or 1h floor). Conservative
	// default — the table is tiny.
	LoginAttemptsCleanupInterval time.Duration

	DB DBConfig
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

// DSN renders the PostgreSQL Data Source Name for opening the
// connection pool with database/sql and the pgx stdlib adapter.
//
// The password is URL-escaped because secrets generated with
// `openssl rand -base64 N` (or any other tool producing
// URL-unsafe characters) contain `+`, `/`, `=` and friends. Without
// escaping, those break the DSN parser with errors like "failed to
// parse as URL (invalid port ":eL4B8" after host)". The user and
// database name are escaped too for the same reason — usernames with
// `@` and database names with reserved characters would otherwise
// silently break. SSLMode is also escaped because future values
// like `verify-full` could one day ship with extra parameters.
func (dbConfig DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.QueryEscape(dbConfig.User),
		url.QueryEscape(dbConfig.Password),
		dbConfig.Host,
		dbConfig.Port,
		url.QueryEscape(dbConfig.Name),
		url.QueryEscape(dbConfig.SSLMode),
	)
}

// LoadConfig reads configuration from environment variables and
// FAILS FAST on any missing or weak secret.
//
// Required in every environment (no fallback defaults — defaults for
// secrets are how an operator ends up running production with a
// publicly-known JWT signing key):
//
//   JWT_SECRET   must be at least 32 bytes (OWASP HS256).
//   DB_PASSWORD  must be at least 12 bytes (OWASP password baseline).
//   ENV          one of: development, staging, production.
//   DB_SSLMODE   one of: disable, allow, prefer, require, verify-ca, verify-full.
//
// Optional with safe defaults (numeric / behavioral, never secrets):
//
//   PORT                 (8080)
//   DB_HOST              (empty — operator must set)
//   DB_PORT              (5432)
//   DB_USER              (empty — operator must set)
//   DB_NAME              (empty — operator must set)
//   DB_MAX_OPEN_CONNS    (25)
//   DB_MAX_IDLE_CONNS    (5)
//   DB_CONN_MAX_LIFETIME (5m)
//   DB_PING_TIMEOUT      (30s)
//   SHUTDOWN_TIMEOUT     (15s)
//   CORS_ORIGINS                 (REQUIRED in production; permissive in development)
//   TRUSTED_PROXIES              (none)
//   TLS_KEY_FILE                 (none)
//   TLS_CERT_FILE                (none)
//   JWT_ACCESS_TTL               (1h)
//   JWT_REFRESH_TTL              (24h)
//   COOKIE_SECURE                (true when TLS or production; false in development)
//   LOGIN_IP_RATE_PER_MINUTE     (20)
//   LOGIN_IP_BURST               (10)
//   REGISTER_IP_RATE_PER_MINUTE  (5)
//   REGISTER_IP_BURST            (5)
//   LOCKOUT_EMAIL_MAX_FAILURES   (5)
//   LOCKOUT_EMAIL_WINDOW         (15m)
//   LOCKOUT_IP_MAX_FAILURES      (10)
//   LOCKOUT_IP_WINDOW            (5m)
//   LOGIN_ATTEMPTS_CLEANUP_INTERVAL (15m)
//
// Note: DB_HOST, DB_USER, DB_NAME have no in-code default on purpose.
// The operator must declare them. A bare "localhost" default is
// misleading: it makes a misconfigured production box silently try
// to connect to its own loopback. Force the operator to spell it out.
func LoadConfig() (Config, error) {
	env, err := loadEnv()
	if err != nil {
		return Config{}, err
	}

	jwtSecret, err := loadJWTSecret()
	if err != nil {
		return Config{}, err
	}

	dbPassword, err := loadDBPassword()
	if err != nil {
		return Config{}, err
	}

	sslMode, err := loadDBSSLMode()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Env:             env,
		Port:            getEnvInt("PORT", 8080),
		JWTSecret:       jwtSecret,
		JWTAccessTTL:    getEnvDuration("JWT_ACCESS_TTL", 1*time.Hour),
		JWTRefreshTTL:   getEnvDuration("JWT_REFRESH_TTL", 24*time.Hour),
		CookieSecure:    resolveCookieSecure(env, os.Getenv("COOKIE_SECURE")),
		CORSOrigins:     parseCSV(os.Getenv("CORS_ORIGINS")),
		TLSKeyFile:      os.Getenv("TLS_KEY_FILE"),
		TLSCertFile:     os.Getenv("TLS_CERT_FILE"),
		TrustedProxies:  parseCSV(os.Getenv("TRUSTED_PROXIES")),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),

		LoginIPRateLimitPerMinute:    getEnvInt("LOGIN_IP_RATE_PER_MINUTE", 20),
		LoginIPBurst:                 getEnvInt("LOGIN_IP_BURST", 10),
		RegisterIPRateLimitPerMinute: getEnvInt("REGISTER_IP_RATE_PER_MINUTE", 5),
		RegisterIPBurst:              getEnvInt("REGISTER_IP_BURST", 5),

		LockoutEmailMaxFailures: getEnvInt("LOCKOUT_EMAIL_MAX_FAILURES", 5),
		LockoutEmailWindow:      getEnvDuration("LOCKOUT_EMAIL_WINDOW", 15*time.Minute),
		LockoutIPMaxFailures:    getEnvInt("LOCKOUT_IP_MAX_FAILURES", 10),
		LockoutIPWindow:         getEnvDuration("LOCKOUT_IP_WINDOW", 5*time.Minute),

		LoginAttemptsCleanupInterval: getEnvDuration("LOGIN_ATTEMPTS_CLEANUP_INTERVAL", 15*time.Minute),

		DB: DBConfig{
			Host:            os.Getenv("DB_HOST"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            os.Getenv("DB_USER"),
			Password:        dbPassword,
			Name:            os.Getenv("DB_NAME"),
			SSLMode:         sslMode,
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			PingTimeout:     getEnvDuration("DB_PING_TIMEOUT", 30*time.Second),
		},
	}
	if cfg.DB.Host == "" {
		return Config{}, fmt.Errorf("DB_HOST is required")
	}
	if cfg.DB.User == "" {
		return Config{}, fmt.Errorf("DB_USER is required")
	}
	if cfg.DB.Name == "" {
		return Config{}, fmt.Errorf("DB_NAME is required")
	}
	// CORS_ORIGINS is mandatory in production. An empty allow-list
	// with AllowCredentials=true makes cookies unreceivable by any
	// browser (every browser rejects wildcard credentials), so the
	// failure mode is "nobody can log in" — better to fail loud at
	// startup than to discover it after the deploy.
	if env == "production" && len(cfg.CORSOrigins) == 0 {
		return Config{}, fmt.Errorf("CORS_ORIGINS is required in production (comma-separated list of allowed origins)")
	}
	return cfg, nil
}

// resolveCookieSecure decides whether session cookies carry the
// Secure attribute. Explicit COOKIE_SECURE env wins. Otherwise:
//   - production: true (HTTP-only deploys are forbidden; if the
//     operator runs without TLS they should set COOKIE_SECURE=false
//     and accept the XSS-over-HTTP downgrade).
//   - development/staging: false (browsers refuse Secure on http://).
func resolveCookieSecure(env, raw string) bool {
	if raw != "" {
		switch raw {
		case "true", "1", "yes":
			return true
		case "false", "0", "no":
			return false
		}
	}
	return env == "production"
}

// loadEnv validates ENV is one of the closed set of accepted modes.
// A typo (e.g. "producton") MUST NOT silently fall through to
// "development" semantics.
func loadEnv() (string, error) {
	env := os.Getenv("ENV")
	if env == "" {
		return "", fmt.Errorf("ENV is required (one of: development, staging, production)")
	}
	if _, ok := validEnvValues[env]; !ok {
		return "", fmt.Errorf("ENV=%q is invalid (must be one of: development, staging, production)", env)
	}
	return env, nil
}

// loadJWTSecret returns the JWT signing key, failing if it is
// missing or shorter than the HS256 entropy floor. The previous
// implementation silently fell back to "dev-secret", which let any
// attacker forge admin tokens if the operator forgot to set ENV.
func loadJWTSecret() (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET is required and must be at least %d bytes", minJWTSecretBytes)
	}
	if len(secret) < minJWTSecretBytes {
		return "", fmt.Errorf("JWT_SECRET is too short: got %d bytes, need at least %d (generate with `openssl rand -base64 48`)", len(secret), minJWTSecretBytes)
	}
	return secret, nil
}

// loadDBPassword returns the database password, failing if it is
// missing or shorter than the OWASP password baseline. The previous
// implementation silently fell back to "dogpaw_pass", which is the
// same string the docker-compose file published to all interfaces.
func loadDBPassword() (string, error) {
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		return "", fmt.Errorf("DB_PASSWORD is required and must be at least %d bytes", minDBPasswordBytes)
	}
	if len(password) < minDBPasswordBytes {
		return "", fmt.Errorf("DB_PASSWORD is too short: got %d bytes, need at least %d", len(password), minDBPasswordBytes)
	}
	return password, nil
}

// loadDBSSLMode validates the Postgres SSL mode against the closed
// set. The previous implementation silently defaulted to "" (which
// the driver interprets as "prefer" — plaintext if the server
// accepts it), making TLS bypassable by configuration drift.
func loadDBSSLMode() (string, error) {
	mode := os.Getenv("DB_SSLMODE")
	if mode == "" {
		return "", fmt.Errorf("DB_SSLMODE is required (one of: disable, allow, prefer, require, verify-ca, verify-full)")
	}
	if _, ok := validSSLModes[mode]; !ok {
		return "", fmt.Errorf("DB_SSLMODE=%q is invalid (must be one of: disable, allow, prefer, require, verify-ca, verify-full)", mode)
	}
	return mode, nil
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
