package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"

	"dogpaw/internal/crypto"
	"dogpaw/migrations"
)

type devUser struct {
	name     string
	email    string
	password string
	role     string
}

// devUsers holds well-known local credentials. ensureDevUsers upserts these
// so a development instance always has a usable admin and demo account.
var devUsers = []devUser{
	{name: "Carlos Admin", email: "admin@dogpaw.com", password: "admin123", role: "ADMIN"},
	{name: "Demo Owner", email: "demo@dogpaw.com", password: "demo1234", role: "REGULAR"},
}

// ensureDevUsers guarantees the development users from devUsers exist with
// the documented passwords. It is a no-op in production to avoid planting
// default credentials in a deployed environment. The returned bool reports
// whether the function actually ran (i.e. we are in a non-production env).
func ensureDevUsers(db *sql.DB, env string) (bool, error) {
	if env == "production" {
		return false, nil
	}

	hasher := crypto.NewDefaultBcryptHasher()

	tx, err := db.Begin()
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, user := range devUsers {
		hashed, err := hasher.Hash(user.password)
		if err != nil {
			return false, fmt.Errorf("hash password for %s: %w", user.email, err)
		}

		var currentPassword, currentRole string
		err = tx.QueryRow(
			`SELECT password, role FROM users WHERE email = $1`, user.email,
		).Scan(&currentPassword, &currentRole)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if _, err := tx.Exec(
				`INSERT INTO users (name, email, password, role, is_active)
				 VALUES ($1, $2, $3, $4, TRUE)`,
				user.name, user.email, hashed, user.role,
			); err != nil {
				return false, fmt.Errorf("insert dev user %s: %w", user.email, err)
			}
		case err != nil:
			return false, fmt.Errorf("lookup dev user %s: %w", user.email, err)
		default:
			if compareErr := hasher.Compare(currentPassword, user.password); compareErr != nil || currentRole != user.role {
				if _, err := tx.Exec(
					`UPDATE users SET password = $1, role = $2, updated_at = NOW() WHERE email = $3`,
					hashed, user.role, user.email,
				); err != nil {
					return false, fmt.Errorf("update dev user %s: %w", user.email, err)
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit tx: %w", err)
	}
	return true, nil
}

func openDB(ctx context.Context, cfg DBConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := pingWithRetry(ctx, db, cfg.PingTimeout); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

func pingWithRetry(ctx context.Context, db *sql.DB, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := time.Second
	for {
		pingCtx, cancel := context.WithDeadline(ctx, deadline)
		err := db.PingContext(pingCtx)
		cancel()
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		slog.Warn("db not ready, retrying", "err", err.Error(), "backoff", backoff.String())
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 8*time.Second {
			backoff *= 2
		}
	}
}

func runMigrations(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}
	defer src.Close()

	driver, err := migratepg.WithInstance(db, &migratepg.Config{})
	if err != nil {
		return fmt.Errorf("postgres driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
