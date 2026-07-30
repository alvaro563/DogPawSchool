package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/crypto"
	"dogpaw/internal/domain"
	"dogpaw/internal/repository/postgres"
)

func TestChangePassword_Integration_Success(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanTables(t, testDB)

	oldPassword := "old-password-123"
	newPassword := "new-secure-password-456"
	seeded := seedActiveUser(t, testDB, "charlie@dogpaw.com", oldPassword, domain.RoleRegular)

	verifier := crypto.NewDefaultBcryptHasher()
	hasher := crypto.NewDefaultBcryptHasher()
	userRepo := postgres.NewUserRepository(testDB)

	loginUC := NewLoginUseCase(userRepo, verifier, crypto.NewJWTTokenGenerator("test-secret", 1*time.Hour))
	changeUC := NewChangePasswordUseCase(userRepo, verifier, hasher)

	// Step 1: Login with old password works
	loginIn := MustNewLoginInput("charlie@dogpaw.com", oldPassword, nil)
	loginOut, err := loginUC.Execute(context.Background(), loginIn)
	require.NoError(t, err, "login with old password must succeed")
	require.NotEmpty(t, loginOut.Token)

	// Step 2: Change password
	changeIn := MustNewChangePasswordInput(seeded.ID(), oldPassword, newPassword, nil)
	_, err = changeUC.Execute(context.Background(), changeIn)
	require.NoError(t, err, "change password must succeed")

	// Step 3: Login with old password fails
	_, err = loginUC.Execute(context.Background(), MustNewLoginInput("charlie@dogpaw.com", oldPassword, nil))
	assert.ErrorIs(t, err, ErrInvalidCredentials, "old password must no longer work")

	// Step 4: Login with new password works
	loginOut2, err := loginUC.Execute(context.Background(), MustNewLoginInput("charlie@dogpaw.com", newPassword, nil))
	require.NoError(t, err, "login with new password must succeed")
	require.NotEmpty(t, loginOut2.Token)

	claims, err := crypto.ParseToken(loginOut2.Token, []byte("test-secret"))
	require.NoError(t, err)
	assert.Equal(t, seeded.ID(), claims.UserID)
}

func TestChangePassword_Integration_UserNotFound(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanTables(t, testDB)

	verifier := crypto.NewDefaultBcryptHasher()
	hasher := crypto.NewDefaultBcryptHasher()
	userRepo := postgres.NewUserRepository(testDB)
	uc := NewChangePasswordUseCase(userRepo, verifier, hasher)
	in := MustNewChangePasswordInput(9999, "old", "new-secure-password", nil)

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestChangePassword_Integration_WrongOldPassword(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanTables(t, testDB)

	seeded := seedActiveUser(t, testDB, "diana@dogpaw.com", "real-old-password", domain.RoleRegular)

	verifier := crypto.NewDefaultBcryptHasher()
	hasher := crypto.NewDefaultBcryptHasher()
	userRepo := postgres.NewUserRepository(testDB)
	uc := NewChangePasswordUseCase(userRepo, verifier, hasher)
	in := MustNewChangePasswordInput(seeded.ID(), "wrong-old-password", "new-secure-password", nil)

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestChangePassword_Integration_SamePassword(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanTables(t, testDB)

	seeded := seedActiveUser(t, testDB, "eve@dogpaw.com", "current-password-123", domain.RoleRegular)

	verifier := crypto.NewDefaultBcryptHasher()
	hasher := crypto.NewDefaultBcryptHasher()
	userRepo := postgres.NewUserRepository(testDB)
	uc := NewChangePasswordUseCase(userRepo, verifier, hasher)
	in := MustNewChangePasswordInput(seeded.ID(), "current-password-123", "current-password-123", nil)

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrSamePassword)
}

func TestChangePassword_Integration_InactiveUser(t *testing.T) {
	if testDB == nil {
		t.Fatal("testDB is nil — TestMain did not run or failed")
	}
	cleanTables(t, testDB)

	seedInactiveUser(t, testDB, "frank@dogpaw.com", "some-password", domain.RoleRegular)

	verifier := crypto.NewDefaultBcryptHasher()
	hasher := crypto.NewDefaultBcryptHasher()
	userRepo := postgres.NewUserRepository(testDB)
	uc := NewChangePasswordUseCase(userRepo, verifier, hasher)
	in := MustNewChangePasswordInput(1, "some-password", "new-secure-password", nil)

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}
