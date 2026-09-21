// @title           DogPaw API
// @version         0.1.0
// @description     API for managing dog care activities, reservations, passes and users. Protected endpoints require a Bearer JWT obtained from the login endpoint.
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description     Paste the token returned by /api/v1/auth/login as "Bearer <token>".
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err.Error())
		os.Exit(1)
	}
}

func run() error {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	// Startup banner. The operator should be able to confirm at a
	// glance which mode the process is running in and whether TLS
	// is configured. NEVER log the secret values themselves — only
	// the metadata above.
	slog.Info("config loaded",
		"env", cfg.Env,
		"port", cfg.Port,
		"db_sslmode", cfg.DB.SSLMode,
		"tls", cfg.TLSCertFile != "" && cfg.TLSKeyFile != "",
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := openDB(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("close db", "err", err.Error())
		}
	}()
	slog.Info("db connected", "host", cfg.DB.Host, "port", cfg.DB.Port, "name", cfg.DB.Name)

	if err := runMigrations(db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	slog.Info("migrations applied")

	router := newRouter(db, cfg)
	return startServer(ctx, cfg, router)
}
