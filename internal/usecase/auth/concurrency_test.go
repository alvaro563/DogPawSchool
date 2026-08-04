package auth

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"dogpaw/internal/crypto"
	"dogpaw/internal/domain"
	"dogpaw/internal/repository/postgres"
	"dogpaw/migrations"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	c, connStr, err := startPostgresContainer(ctx)
	if err != nil {
		log.Fatalf("auth concurrency: start container: %v", err)
	}
	defer func() {
		if err := c.Terminate(ctx); err != nil {
			log.Printf("auth concurrency: terminate container: %v", err)
		}
	}()

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("auth concurrency: sql.Open: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("auth concurrency: db.Ping: %v", err)
	}

	if err := runMigrations(db); err != nil {
		log.Fatalf("auth concurrency: migrate: %v", err)
	}

	testDB = db
	os.Exit(m.Run())
}

func startPostgresContainer(ctx context.Context) (*tcpostgres.PostgresContainer, string, error) {
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
		return nil, "", err
	}
	connStr, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, "", err
	}
	return c, connStr, nil
}

func runMigrations(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	defer src.Close()
	driver, err := migratepg.WithInstance(db, &migratepg.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func seedPendingInvitation(t *testing.T, db *sql.DB, token string) *domain.Invitation {
	t.Helper()

	// Create an admin user first (FK: invitations.created_by).
	adminPw := make([]byte, 60)
	for i := range adminPw {
		adminPw[i] = 'a'
	}
	admin, err := domain.NewUser(0, "Admin", "admin@test.com", string(adminPw), domain.RoleAdmin)
	require.NoError(t, err)
	userRepo := postgres.NewUserRepository(db)
	_, err = userRepo.Create(context.Background(), admin)
	require.NoError(t, err)
	admin, err = userRepo.GetByEmail(context.Background(), "admin@test.com")
	require.NoError(t, err)

	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(48 * time.Hour)
	inv, err := domain.NewPendingInvitation(admin.ID(), "client@test.com", hashRegistrationToken(token), domain.RoleRegular, expiresAt, now)
	require.NoError(t, err)
	invRepo := postgres.NewInvitationRepository(db)
	id, err := invRepo.Create(context.Background(), inv)
	require.NoError(t, err)
	require.Positive(t, id)

	created, err := invRepo.GetByToken(context.Background(), hashRegistrationToken(token))
	require.NoError(t, err)
	require.NotNil(t, created)
	return created
}

// TestRegisterWithInvitation_ConcurrentTokenUse validates that the
// SELECT ... FOR UPDATE row lock serialises concurrent registration
// attempts with the same invitation token. Only one goroutine should
// succeed; all others must fail with ErrInvitationInvalid or
// ErrDuplicateEmail.
func TestRegisterWithInvitation_ConcurrentTokenUse(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}

	token := "aaaabbbbccccddddeeeeffff0000111122223333444455556666777788889999"
	password := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)

	// Clean + seed.
	cleanTables(t, testDB)
	_ = seedPendingInvitation(t, testDB, token)

	// Real dependencies.
	invRepo := postgres.NewInvitationRepository(testDB)
	userRepo := postgres.NewUserRepository(testDB)
	transactor := postgres.NewTransactor(testDB)
	hasher := crypto.NewDefaultBcryptHasher()

	uc := NewRegisterWithInvitationUseCase(transactor, invRepo, userRepo, hasher)

	const goroutines = 20
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		successes int
		errors_   int
	)

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			in := MustNewRegisterWithInvitationInput(token, "Client", password, func() time.Time { return now })
			_, err := uc.Execute(context.Background(), in)
			mu.Lock()
			if err == nil {
				successes++
			} else {
				errors_++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	require.Equal(t, 1, successes, "exactly one registration must succeed")
	require.Equal(t, goroutines-1, errors_, "all other goroutines must fail")

	// Verify the invitation is now ACCEPTED in the database.
	invRepo2 := postgres.NewInvitationRepository(testDB)
	inv, err := invRepo2.GetByToken(context.Background(), hashRegistrationToken(token))
	require.NoError(t, err)
	require.NotNil(t, inv)
	require.Equal(t, domain.InvitationAccepted, inv.Status())
}

func cleanTables(t *testing.T, db *sql.DB) {
	t.Helper()
	tables := []string{
		"pass_movements", "reservations", "invitations",
		"dog_incompatibilities", "passes", "dogs",
		"activities", "incompatibilities", "users",
	}
	_, err := db.ExecContext(context.Background(),
		"TRUNCATE TABLE "+joinStrings(tables, ",")+" RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func joinStrings(elems []string, sep string) string {
	out := ""
	for i, e := range elems {
		if i > 0 {
			out += sep
		}
		out += e
	}
	return out
}

func TestRegisterWithInvitation_Integration(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}

	token := "integration-test-token-00112233445566778899aabbccddeeff"
	password := "securepassword123"
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)

	cleanTables(t, testDB)
	inv := seedPendingInvitation(t, testDB, token)

	invRepo := postgres.NewInvitationRepository(testDB)
	userRepo := postgres.NewUserRepository(testDB)
	transactor := postgres.NewTransactor(testDB)
	hasher := crypto.NewDefaultBcryptHasher()

	uc := NewRegisterWithInvitationUseCase(transactor, invRepo, userRepo, hasher)
	in := MustNewRegisterWithInvitationInput(token, "Client Name", password, func() time.Time { return now })

	out, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)
	require.NotNil(t, out.User)

	require.Positive(t, out.User.ID(), "user must have a positive id")
	assert.Equal(t, "Client Name", out.User.Name())
	assert.Equal(t, inv.Email(), out.User.Email())
	assert.Equal(t, inv.Role(), out.User.Role())
	assert.True(t, out.User.IsActive())
	assert.NotContains(t, out.User.Password(), password, "must not contain plaintext")

	created, err := userRepo.GetByID(context.Background(), out.User.ID())
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "Client Name", created.Name())
	assert.Equal(t, inv.Email(), created.Email())
	assert.Equal(t, inv.Role(), created.Role())
	assert.True(t, created.IsActive())
	assert.Regexp(t, `^\$2[abxy]\$`, created.Password(), "password must be a bcrypt hash")

	storedInv, err := invRepo.GetByToken(context.Background(), hashRegistrationToken(token))
	require.NoError(t, err)
	require.NotNil(t, storedInv)
	assert.Equal(t, domain.InvitationAccepted, storedInv.Status(), "invitation must be ACCEPTED")
}
