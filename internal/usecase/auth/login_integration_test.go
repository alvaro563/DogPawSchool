package auth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/crypto"
	"dogpaw/internal/domain"
	"dogpaw/internal/repository/postgres"
)

func seedActiveUser(t *testing.T, db *sql.DB, email, plainPassword string, role domain.UserRole) *domain.User {
	t.Helper()
	hasher := crypto.NewDefaultBcryptHasher()
	hashed, err := hasher.Hash(plainPassword)
	require.NoError(t, err)
	user, err := domain.NewUser(0, "Test User", email, hashed, role)
	require.NoError(t, err)
	userRepo := postgres.NewUserRepository(db)
	userID, err := userRepo.Create(context.Background(), user)
	require.NoError(t, err)
	require.Positive(t, userID)
	user, err = domain.NewUser(userID, user.Name(), user.Email(), user.Password(), user.Role())
	require.NoError(t, err)
	return user
}

func seedInactiveUser(t *testing.T, db *sql.DB, email, plainPassword string, role domain.UserRole) *domain.User {
	t.Helper()
	user := seedActiveUser(t, db, email, plainPassword, role)
	user.Deactivate()
	userRepo := postgres.NewUserRepository(db)
	err := userRepo.Update(context.Background(), user)
	require.NoError(t, err)
	return user
}

func TestLogin_Integration_Success(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanTables(t, testDB)

	plainPassword := "secure-password-123"
	seeded := seedActiveUser(t, testDB, "alice@dogpaw.com", plainPassword, domain.RoleRegular)

	verifier := crypto.NewDefaultBcryptHasher()
	tokenGen := crypto.NewJWTTokenGenerator("test-secret", 1*time.Hour)
	userRepo := postgres.NewUserRepository(testDB)
	uc := NewLoginUseCase(userRepo, verifier, tokenGen)
	in := MustNewLoginInput("alice@dogpaw.com", plainPassword, nil)

	out, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)
	require.NotEmpty(t, out.Token, "token must not be empty")
	require.NotNil(t, out.User)
	assert.Equal(t, seeded.ID(), out.User.ID())
	assert.Equal(t, "alice@dogpaw.com", out.User.Email())

	claims, err := crypto.ParseToken(out.Token, []byte("test-secret"))
	require.NoError(t, err, "token must be parseable")
	assert.Equal(t, seeded.ID(), claims.UserID)
	assert.Equal(t, "REGULAR", claims.Role)
}

func TestLogin_Integration_EmailNotFound(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanTables(t, testDB)

	verifier := crypto.NewDefaultBcryptHasher()
	tokenGen := crypto.NewJWTTokenGenerator("test-secret", 1*time.Hour)
	userRepo := postgres.NewUserRepository(testDB)
	uc := NewLoginUseCase(userRepo, verifier, tokenGen)
	in := MustNewLoginInput("nonexistent@dogpaw.com", "any-password", nil)

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_Integration_WrongPassword(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanTables(t, testDB)

	seedActiveUser(t, testDB, "bob@dogpaw.com", "correct-password", domain.RoleRegular)

	verifier := crypto.NewDefaultBcryptHasher()
	tokenGen := crypto.NewJWTTokenGenerator("test-secret", 1*time.Hour)
	userRepo := postgres.NewUserRepository(testDB)
	uc := NewLoginUseCase(userRepo, verifier, tokenGen)
	in := MustNewLoginInput("bob@dogpaw.com", "wrong-password", nil)

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_Integration_InactiveUser(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanTables(t, testDB)

	seedInactiveUser(t, testDB, "deactivated@dogpaw.com", "some-password", domain.RoleRegular)

	verifier := crypto.NewDefaultBcryptHasher()
	tokenGen := crypto.NewJWTTokenGenerator("test-secret", 1*time.Hour)
	userRepo := postgres.NewUserRepository(testDB)
	uc := NewLoginUseCase(userRepo, verifier, tokenGen)
	in := MustNewLoginInput("deactivated@dogpaw.com", "some-password", nil)

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}
