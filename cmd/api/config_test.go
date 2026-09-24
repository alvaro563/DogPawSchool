package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setRequiredEnv sets every required env var to a known-valid value
// for tests that need a baseline "happy config" to override. Centralised
// here so adding a new required var only touches one place.
func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ENV", "development")
	t.Setenv("JWT_SECRET", "test-secret-32-bytes-of-entropy!!")
	t.Setenv("DB_PASSWORD", "test-db-password-12+chars")
	t.Setenv("DB_SSLMODE", "disable")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_USER", "dogpaw_user")
	t.Setenv("DB_NAME", "dogpaw_db")
}

func TestLoadConfig_HappyPath(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "test-secret-32-bytes-of-entropy!!", cfg.JWTSecret)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, 15*time.Second, cfg.ShutdownTimeout)
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "dogpaw_user", cfg.DB.User)
	assert.Equal(t, "test-db-password-12+chars", cfg.DB.Password)
	assert.Equal(t, "dogpaw_db", cfg.DB.Name)
	assert.Equal(t, "disable", cfg.DB.SSLMode)
	assert.Equal(t, 25, cfg.DB.MaxOpenConns)
	assert.Equal(t, 5, cfg.DB.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.DB.ConnMaxLifetime)
	assert.Equal(t, 30*time.Second, cfg.DB.PingTimeout)
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ENV", "production")
	t.Setenv("PORT", "3000")
	t.Setenv("SHUTDOWN_TIMEOUT", "30s")
	t.Setenv("DB_HOST", "pg.internal")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "admin")
	t.Setenv("DB_PASSWORD", "prod-db-password-12+chars")
	t.Setenv("DB_NAME", "analytics")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("JWT_SECRET", "prod-jwt-secret-32-bytes-or-more!")
	t.Setenv("CORS_ORIGINS", "https://app.example.com")
	t.Setenv("DB_MAX_OPEN_CONNS", "50")
	t.Setenv("DB_MAX_IDLE_CONNS", "10")
	t.Setenv("DB_CONN_MAX_LIFETIME", "10m")
	t.Setenv("DB_PING_TIMEOUT", "15s")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "prod-jwt-secret-32-bytes-or-more!", cfg.JWTSecret)
	assert.Equal(t, 3000, cfg.Port)
	assert.Equal(t, 30*time.Second, cfg.ShutdownTimeout)
	assert.Equal(t, "pg.internal", cfg.DB.Host)
	assert.Equal(t, 5433, cfg.DB.Port)
	assert.Equal(t, "admin", cfg.DB.User)
	assert.Equal(t, "prod-db-password-12+chars", cfg.DB.Password)
	assert.Equal(t, "analytics", cfg.DB.Name)
	assert.Equal(t, "require", cfg.DB.SSLMode)
	assert.Equal(t, 50, cfg.DB.MaxOpenConns)
	assert.Equal(t, 10, cfg.DB.MaxIdleConns)
	assert.Equal(t, 10*time.Minute, cfg.DB.ConnMaxLifetime)
	assert.Equal(t, 15*time.Second, cfg.DB.PingTimeout)
}

func TestLoadConfig_ProductionRequiresCORSOrigins(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ENV", "production")
	// CORS_ORIGINS intentionally not set.
	_, err := LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CORS_ORIGINS is required in production")
}

func TestLoadConfig_DevelopmentCORSOriginsOptional(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ENV", "development")
	// Empty CORS_ORIGINS is fine in development.
	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Empty(t, cfg.CORSOrigins, "development defaults to permissive CORS (empty allowlist)")
}

// TestLoadConfig_DevRequiresSecretsToo is the anti-regression test
// for the original bug. Before this commit, an unset JWT_SECRET in
// a "development" environment silently fell back to "dev-secret",
// letting any attacker forge admin tokens. A development
// environment must demand real secrets too — there is no safe mode.
func TestLoadConfig_DevRequiresSecretsToo(t *testing.T) {
	t.Run("missing_jwt_secret_in_dev_fails", func(t *testing.T) {
		t.Setenv("ENV", "development")
		t.Setenv("DB_PASSWORD", "test-db-password-12+chars")
		t.Setenv("DB_SSLMODE", "disable")
		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_USER", "dogpaw_user")
		t.Setenv("DB_NAME", "dogpaw_db")

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_SECRET")
		assert.NotContains(t, err.Error(), "dev-secret", "must NEVER mention the old fallback")
	})

	t.Run("missing_db_password_in_dev_fails", func(t *testing.T) {
		t.Setenv("ENV", "development")
		t.Setenv("JWT_SECRET", "test-secret-32-bytes-of-entropy!!")
		t.Setenv("DB_SSLMODE", "disable")
		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_USER", "dogpaw_user")
		t.Setenv("DB_NAME", "dogpaw_db")

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_PASSWORD")
		assert.NotContains(t, err.Error(), "dogpaw_pass", "must NEVER mention the old fallback")
	})

	t.Run("missing_env_var_fails", func(t *testing.T) {
		// Unset everything we care about and verify ENV itself is
		// required.
		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ENV")
	})

	t.Run("missing_db_sslmode_fails", func(t *testing.T) {
		t.Setenv("ENV", "development")
		t.Setenv("JWT_SECRET", "test-secret-32-bytes-of-entropy!!")
		t.Setenv("DB_PASSWORD", "test-db-password-12+chars")
		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_USER", "dogpaw_user")
		t.Setenv("DB_NAME", "dogpaw_db")
		// DB_SSLMODE intentionally unset

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_SSLMODE")
	})
}

func TestLoadConfig_StrongSecretRequired(t *testing.T) {
	t.Run("jwt_secret_too_short_fails", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("JWT_SECRET", "short")

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_SECRET")
		assert.Contains(t, err.Error(), "32")
	})

	t.Run("jwt_secret_at_exactly_32_succeeds", func(t *testing.T) {
		setRequiredEnv(t)
		// 31 chars — must fail.
		t.Setenv("JWT_SECRET", strings.Repeat("a", 31))
		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_SECRET")

		// 32 chars — must succeed.
		t.Setenv("JWT_SECRET", strings.Repeat("a", 32))
		_, err = LoadConfig()
		require.NoError(t, err)
	})

	t.Run("db_password_too_short_fails", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("DB_PASSWORD", "short")

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_PASSWORD")
		assert.Contains(t, err.Error(), "12")
	})
}

func TestLoadConfig_ClosedEnvValueValidation(t *testing.T) {
	t.Run("invalid_env_value_fails", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("ENV", "producton") // typo

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ENV")
		assert.Contains(t, err.Error(), "producton")
	})

	t.Run("all_three_valid_env_values_succeed", func(t *testing.T) {
		for _, env := range []string{"development", "staging", "production"} {
			t.Run(env, func(t *testing.T) {
				setRequiredEnv(t)
				t.Setenv("ENV", env)
				if env == "production" {
					// Required since the CORS hardening in this
					// commit: an empty AllowOrigins with
					// AllowCredentials=true makes cookies
					// unreceivable by any browser.
					t.Setenv("CORS_ORIGINS", "https://app.example.com")
				}

				cfg, err := LoadConfig()
				require.NoError(t, err)
				assert.Equal(t, env, cfg.Env)
			})
		}
	})

	t.Run("invalid_db_sslmode_fails", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("DB_SSLMODE", "banana")

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_SSLMODE")
		assert.Contains(t, err.Error(), "banana")
	})

	t.Run("all_six_valid_sslmodes_succeed", func(t *testing.T) {
		for _, mode := range []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"} {
			t.Run(mode, func(t *testing.T) {
				setRequiredEnv(t)
				t.Setenv("DB_SSLMODE", mode)

				cfg, err := LoadConfig()
				require.NoError(t, err)
				assert.Equal(t, mode, cfg.DB.SSLMode)
			})
		}
	})
}

func TestLoadConfig_MissingDBIdentifiers(t *testing.T) {
	// DB_HOST, DB_USER, DB_NAME have no in-code defaults on purpose.
	// A misconfigured operator must see a clear error rather than a
	// silent "localhost" fallback that masks the problem.
	setRequiredEnv(t)
	t.Setenv("DB_HOST", "")
	_, err := LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_HOST")

	setRequiredEnv(t)
	t.Setenv("DB_USER", "")
	_, err = LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_USER")

	setRequiredEnv(t)
	t.Setenv("DB_NAME", "")
	_, err = LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_NAME")
}

func TestLoadConfig_NumericFieldsFallBack(t *testing.T) {
	// Numeric / behavioral fields still fall back to sensible defaults
	// when their env var is missing or malformed. Only secrets fail
	// fast.
	setRequiredEnv(t)
	t.Setenv("PORT", "not-a-number")
	t.Setenv("DB_PORT", "also-bad")
	t.Setenv("DB_CONN_MAX_LIFETIME", "never")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, 5*time.Minute, cfg.DB.ConnMaxLifetime)
}

func TestDBConfig_DSN(t *testing.T) {
	t.Parallel()

	t.Run("standard_format", func(t *testing.T) {
		t.Parallel()
		cfg := DBConfig{
			User: "u", Password: "p", Host: "localhost",
			Port: 5432, Name: "db", SSLMode: "disable",
		}
		assert.Equal(t, "postgres://u:p@localhost:5432/db?sslmode=disable", cfg.DSN())
	})

	t.Run("sslmode_require", func(t *testing.T) {
		t.Parallel()
		cfg := DBConfig{
			User: "admin", Password: "secret", Host: "db.example.com",
			Port: 5432, Name: "proddb", SSLMode: "require",
		}
		assert.Equal(t, "postgres://admin:secret@db.example.com:5432/proddb?sslmode=require", cfg.DSN())
	})

	t.Run("special_chars_in_password_are_escaped", func(t *testing.T) {
		t.Parallel()
		// The raw password "p@ss:w%rd" contains `@` (would terminate
		// the userinfo section), `:` (would terminate the password)
		// and `%` (would start a percent-encoded sequence). Without
		// escaping, the DSN parser would either reject the URL or
		// silently mangle the credentials.
		//
		// `url.QueryEscape` percent-encodes everything except
		// unreserved characters: A-Z a-z 0-9 - _ . ~ . All three
		// offending characters get escaped:
		//   p@ss:w%rd → p%40ss%3Aw%25rd
		cfg := DBConfig{
			User: "user", Password: "p@ss:w%rd", Host: "h",
			Port: 5432, Name: "db", SSLMode: "disable",
		}
		assert.Equal(t, "postgres://user:p%40ss%3Aw%25rd@h:5432/db?sslmode=disable", cfg.DSN())
	})

	t.Run("base64_secrets_are_escaped", func(t *testing.T) {
		t.Parallel()
		// `openssl rand -base64 N` produces a string with `+`, `/`,
		// and trailing `=`. Without escaping these break the DSN
		// parser ("invalid port :eL4B8 after host" was the
		// motivating bug). Verify the escaping is correct for a
		// representative base64 string.
		cfg := DBConfig{
			User: "dogpaw_user", Password: "abc+def/ghi=", Host: "localhost",
			Port: 5432, Name: "dogpaw_db", SSLMode: "disable",
		}
		assert.Equal(t, "postgres://dogpaw_user:abc%2Bdef%2Fghi%3D@localhost:5432/dogpaw_db?sslmode=disable", cfg.DSN())
	})

	t.Run("user_with_at_sign_is_escaped", func(t *testing.T) {
		t.Parallel()
		// Belt-and-braces: usernames can theoretically contain
		// characters reserved by the URL syntax.
		cfg := DBConfig{
			User: "ops@corp", Password: "secret", Host: "h",
			Port: 5432, Name: "db", SSLMode: "disable",
		}
		assert.Equal(t, "postgres://ops%40corp:secret@h:5432/db?sslmode=disable", cfg.DSN())
	})
}
