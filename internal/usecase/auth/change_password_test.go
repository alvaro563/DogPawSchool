package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

// --- Helpers ---

func changePasswordFixedNow() time.Time {
	return time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
}

func changePasswordActiveUser() *domain.User {
	u, err := domain.NewUser(42, "Alice", "alice@dogpaw.com", "hashed-old-pw-60chars-xxxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleRegular)
	if err != nil {
		panic(err)
	}
	return u
}

func changePasswordInactiveUser() *domain.User {
	u := changePasswordActiveUser()
	u.Deactivate()
	return u
}

// --- Input validation tests ---

func TestNewChangePasswordInput(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name          string
		factory       func() (ChangePasswordInput, error)
		expectedField string
	}{
		{
			name: "zero_user_id",
			factory: func() (ChangePasswordInput, error) {
				return NewChangePasswordInput(0, "old", "newpassword", func() time.Time { return changePasswordFixedNow() })
			},
			expectedField: "user_id",
		},
		{
			name: "empty_old_password",
			factory: func() (ChangePasswordInput, error) {
				return NewChangePasswordInput(42, "", "newpassword", func() time.Time { return changePasswordFixedNow() })
			},
			expectedField: "old_password",
		},
		{
			name: "short_new_password",
			factory: func() (ChangePasswordInput, error) {
				return NewChangePasswordInput(42, "oldpw", "short", func() time.Time { return changePasswordFixedNow() })
			},
			expectedField: "new_password",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			_, err := scenario.factory()
			require.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr))
			assert.Equal(t, scenario.expectedField, verr.Field)
		})
	}
}

func TestMustNewChangePasswordInput_panics_on_validation_error(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		MustNewChangePasswordInput(0, "old", "newpassword", func() time.Time { return changePasswordFixedNow() })
	})
}

// --- Execute tests ---

func TestChangePassword_Success(t *testing.T) {
	t.Parallel()

	activeUser := changePasswordActiveUser()
	oldHash := activeUser.Password()

	userRepo := &mockUserRepository{
		getByID: func(_ context.Context, id int) (*domain.User, error) {
			assert.Equal(t, 42, id)
			return activeUser, nil
		},
		update: func(_ context.Context, u *domain.User) error {
			assert.NotEqual(t, oldHash, u.Password(), "password must be updated")
			assert.Equal(t, changePasswordFixedNow(), u.UpdatedAt(), "updatedAt must be bumped")
			return nil
		},
	}
	verifier := &stubPasswordVerifier{
		verify: func(hash, plain string) error {
			if plain == "new-secure-password" {
				return errors.New("different hash")
			}
			return nil
		},
	}
	hasher := &stubHasher{
		hash: func(plain string) (string, error) {
			assert.Equal(t, "new-secure-password", plain)
			return "hashed:new-secure-password", nil
		},
	}

	uc := NewChangePasswordUseCase(userRepo, verifier, hasher)
	in := MustNewChangePasswordInput(42, "old-password", "new-secure-password", func() time.Time { return changePasswordFixedNow() })

	_, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)
}

func TestChangePassword_UserNotFound(t *testing.T) {
	t.Parallel()

	userRepo := &mockUserRepository{
		getByID: func(_ context.Context, _ int) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := NewChangePasswordUseCase(userRepo, &stubPasswordVerifier{}, &stubHasher{})
	in := MustNewChangePasswordInput(99, "old", "new-secure-password", func() time.Time { return changePasswordFixedNow() })

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	t.Parallel()

	userRepo := &mockUserRepository{
		getByID: func(_ context.Context, _ int) (*domain.User, error) {
			return changePasswordActiveUser(), nil
		},
	}
	verifier := &stubPasswordVerifier{
		verify: func(_, plain string) error {
			if plain == "wrong-old-password" {
				return errors.New("bcrypt compare: mismatch")
			}
			return nil
		},
	}
	uc := NewChangePasswordUseCase(userRepo, verifier, &stubHasher{})
	in := MustNewChangePasswordInput(42, "wrong-old-password", "new-secure-password", func() time.Time { return changePasswordFixedNow() })

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestChangePassword_SamePassword(t *testing.T) {
	t.Parallel()

	activeUser := changePasswordActiveUser()

	userRepo := &mockUserRepository{
		getByID: func(_ context.Context, _ int) (*domain.User, error) {
			return activeUser, nil
		},
	}
	verifier := &stubPasswordVerifier{
		verify: func(_, _ string) error {
			return nil
		},
	}
	uc := NewChangePasswordUseCase(userRepo, verifier, &stubHasher{})
	in := MustNewChangePasswordInput(42, "old-password", "same-as-old-password", func() time.Time { return changePasswordFixedNow() })

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrSamePassword)
}

func TestChangePassword_InactiveUser(t *testing.T) {
	t.Parallel()

	userRepo := &mockUserRepository{
		getByID: func(_ context.Context, _ int) (*domain.User, error) {
			return changePasswordInactiveUser(), nil
		},
	}
	uc := NewChangePasswordUseCase(userRepo, &stubPasswordVerifier{}, &stubHasher{})
	in := MustNewChangePasswordInput(42, "old-password", "new-secure-password", func() time.Time { return changePasswordFixedNow() })

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestChangePassword_HasherFailure(t *testing.T) {
	t.Parallel()

	hashErr := errors.New("bcrypt unavailable")
	userRepo := &mockUserRepository{
		getByID: func(_ context.Context, _ int) (*domain.User, error) {
			return changePasswordActiveUser(), nil
		},
	}
	verifier := &stubPasswordVerifier{
		verify: func(_, plain string) error {
			if plain == "new-secure-password" {
				return errors.New("different hash")
			}
			return nil
		},
	}
	hasher := &stubHasher{
		hash: func(string) (string, error) {
			return "", hashErr
		},
	}
	uc := NewChangePasswordUseCase(userRepo, verifier, hasher)
	in := MustNewChangePasswordInput(42, "old-password", "new-secure-password", func() time.Time { return changePasswordFixedNow() })

	_, err := uc.Execute(context.Background(), in)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, hashErr), "must wrap original hasher error")
}

func TestChangePassword_RepositoryFailure(t *testing.T) {
	t.Parallel()

	updateErr := errors.New("connection refused")
	userRepo := &mockUserRepository{
		getByID: func(_ context.Context, _ int) (*domain.User, error) {
			return changePasswordActiveUser(), nil
		},
		update: func(_ context.Context, _ *domain.User) error {
			return updateErr
		},
	}
	verifier := &stubPasswordVerifier{
		verify: func(_, plain string) error {
			if plain == "new-secure-password" {
				return errors.New("different hash")
			}
			return nil
		},
	}
	uc := NewChangePasswordUseCase(userRepo, verifier, &stubHasher{})
	in := MustNewChangePasswordInput(42, "old-password", "new-secure-password", func() time.Time { return changePasswordFixedNow() })

	_, err := uc.Execute(context.Background(), in)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, updateErr), "must wrap original repository error")
}
