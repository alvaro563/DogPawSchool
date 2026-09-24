package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"dogpaw/internal/crypto"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	c, err := tcpostgres.Run(ctx,
		"postgres:15-alpine",
		tcpostgres.WithDatabase("dogpaw_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategyAndDeadline(
			60*time.Second,
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		log.Fatalf("start container: %v", err)
	}
	defer func() {
		if err := c.Terminate(ctx); err != nil {
			log.Printf("terminate container: %v", err)
		}
	}()

	connStr, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("connection string: %v", err)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("sql.Open: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("db.Ping: %v", err)
	}
	if err := runMigrations(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	testDB = db
	os.Exit(m.Run())
}

func TestMigrations_CreatesAdminUser(t *testing.T) {
	hasher := crypto.NewDefaultBcryptHasher()

	var (
		name     string
		password string
		role     string
		isActive bool
	)

	err := testDB.QueryRow(
		`SELECT name, password, role, is_active FROM users WHERE email = $1`,
		"admin@admin.com",
	).Scan(&name, &password, &role, &isActive)

	require.NoError(t, err, "el usuario administrador debe existir tras las migraciones")
	assert.Equal(t, "Administrador", name)
	assert.Equal(t, "ADMIN", role)
	assert.True(t, isActive)

	// Comprueba que la contraseña encriptada en la migración corresponda a 'elcoledeperrosmola'
	require.NoError(t, hasher.Compare(password, "elcoledeperrosmola"), "la contraseña debe coincidir con 'elcoledeperrosmola'")
}