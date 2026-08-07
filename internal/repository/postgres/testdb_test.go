package postgres

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanTables(t, testDB)
	return testDB
}

func cleanTables(t *testing.T, db *sql.DB) {
	t.Helper()
	tables := []string{
		"pass_movements",
		"reservations",
		"invitations",
		"dog_incompatibilities",
		"passes",
		"dogs",
		"activities",
		"incompatibilities",
		"users",
	}
	_, err := db.ExecContext(context.Background(),
		"TRUNCATE TABLE "+strings.Join(tables, ",")+" RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func insertBaseUser(t *testing.T, db *sql.DB) *domain.User {
	t.Helper()
	password := strings.Repeat("a", 60)
	user, err := domain.NewUser(0, "Test Owner", "owner@test.com", password, domain.RoleAdmin)
	require.NoError(t, err)
	repo := NewUserRepository(db)
	_, err = repo.Create(context.Background(), user)
	require.NoError(t, err)
	got, err := repo.GetByEmail(context.Background(), "owner@test.com")
	require.NoError(t, err)
	require.NotNil(t, got)
	return got
}

func insertBaseActivity(t *testing.T, db *sql.DB) *domain.Activity {
	t.Helper()
	repo := NewActivityRepository(db)
	activity, err := domain.NewActivity(0, "Paseo Test", "", "Parque Central",
		domain.TypeRoute, 10, 1, time.Now().Add(7*24*time.Hour))
	require.NoError(t, err)
	id, err := repo.Create(context.Background(), activity)
	require.NoError(t, err)
	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, id, got.ID())
	return got
}

func insertBaseDog(t *testing.T, db *sql.DB, userID int) *domain.Dog {
	t.Helper()
	repo := NewDogRepository(db)
	dog, err := domain.NewDog(0, "Luna", "Labrador", "ES-TEST-001", 24,
		domain.SexFemale, 22.5, userID)
	require.NoError(t, err)
	id, err := repo.Create(context.Background(), dog)
	require.NoError(t, err)
	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	return got
}

func insertBasePass(t *testing.T, db *sql.DB, userID int) *domain.Pass {
	t.Helper()
	now := time.Now().UTC()
	pass, err := domain.NewPass(0, 10, 10, 5000, domain.PassGeneric, userID, now, now, nil)
	require.NoError(t, err)
	repo := NewPassRepository(db)
	id, err := repo.Create(context.Background(), pass)
	require.NoError(t, err)
	got, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	return got
}

func insertBaseIncompatibility(t *testing.T, db *sql.DB) *domain.Incompatibility {
	t.Helper()
	repo := NewIncompatibilityRepository(db)
	incomp, err := domain.NewTriggerIncompatibility(0, "Reactivo a machos enteros", domain.IncompatibilityLevelAbsoluta, "MACHO_ENTERO")
	require.NoError(t, err)
	id, err := repo.Create(context.Background(), incomp)
	require.NoError(t, err)
	got, err := repo.GetIncompatibilityByID(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, got)
	return got
}


func assertRowCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()
	var count int
	err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM "+table).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, expected, count, "unexpected row count in %s", table)
}

func repeatedString(char string, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = char[0]
	}
	return string(b)
}
