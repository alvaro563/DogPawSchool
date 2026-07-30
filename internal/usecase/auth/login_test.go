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

// --- Stubs for LoginUseCase dependencies ---

type stubPasswordVerifier struct {
	verify func(hash, plain string) error
}

func (s *stubPasswordVerifier) Verify(hash, plain string) error {
	if s.verify != nil {
		return s.verify(hash, plain)
	}
	return nil
}

type stubTokenGenerator struct {
	generate func(user *domain.User) (string, error)
}

func (s *stubTokenGenerator) Generate(user *domain.User) (string, error) {
	if s.generate != nil {
		return s.generate(user)
	}
	return "signed-token", nil
}

// --- Helpers ---

func fixedNowFunc() func() time.Time {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return now }
}

func loginActiveUser() *domain.User {
	u, err := domain.NewUser(42, "Alice", "alice@dogpaw.com", "hashed-60chars-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleRegular)
	if err != nil {
		panic(err)
	}
	return u
}

func loginInactiveUser() *domain.User {
	u := loginActiveUser()
	u.Deactivate()
	return u
}

// --- Input validation tests ---

func TestNewLoginInput(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name          string
		factory       func() (LoginInput, error)
		expectedField string
	}{
		{
			name: "empty_email",
			factory: func() (LoginInput, error) {
				return NewLoginInput("", "secret", fixedNowFunc())
			},
			expectedField: "email",
		},
		{
			name: "empty_password",
			factory: func() (LoginInput, error) {
				return NewLoginInput("alice@dogpaw.com", "", fixedNowFunc())
			},
			expectedField: "password",
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

func TestMustNewLoginInput_panics_on_validation_error(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		MustNewLoginInput("", "pw", fixedNowFunc())
	})
}

// --- Execute tests ---

func TestLogin_Success(t *testing.T) {
	t.Parallel()

	activeUser := loginActiveUser()
	userRepo := &mockUserRepository{
		getByEmail: func(_ context.Context, email string) (*domain.User, error) {
			assert.Equal(t, "alice@dogpaw.com", email)
			return activeUser, nil
		},
	}
	verifier := &stubPasswordVerifier{
		verify: func(hash, plain string) error {
			assert.Equal(t, activeUser.Password(), hash)
			assert.Equal(t, "correct-password", plain)
			return nil
		},
	}
	var capturedUser *domain.User
	tokenGen := &stubTokenGenerator{
		generate: func(user *domain.User) (string, error) {
			capturedUser = user
			return "jwt-header.payload.signature", nil
		},
	}

	uc := NewLoginUseCase(userRepo, verifier, tokenGen)
	in := MustNewLoginInput("alice@dogpaw.com", "correct-password", fixedNowFunc())

	out, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)

	assert.Equal(t, "jwt-header.payload.signature", out.Token)
	assert.Equal(t, activeUser, out.User)
	assert.Equal(t, activeUser, capturedUser, "token generator must receive the authenticated user")
}

func TestLogin_EmailNotFound(t *testing.T) {
	t.Parallel()

	userRepo := &mockUserRepository{
		getByEmail: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := NewLoginUseCase(userRepo, &stubPasswordVerifier{}, &stubTokenGenerator{})
	in := MustNewLoginInput("unknown@dogpaw.com", "any-password", fixedNowFunc())

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_WrongPassword(t *testing.T) {
	t.Parallel()

	userRepo := &mockUserRepository{
		getByEmail: func(_ context.Context, _ string) (*domain.User, error) {
			return loginActiveUser(), nil
		},
	}
	verifier := &stubPasswordVerifier{
		verify: func(_, _ string) error {
			return errors.New("bcrypt compare: mismatch")
		},
	}
	uc := NewLoginUseCase(userRepo, verifier, &stubTokenGenerator{})
	in := MustNewLoginInput("alice@dogpaw.com", "wrong-password", fixedNowFunc())

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_InactiveUser(t *testing.T) {
	t.Parallel()

	userRepo := &mockUserRepository{
		getByEmail: func(_ context.Context, _ string) (*domain.User, error) {
			return loginInactiveUser(), nil
		},
	}
	uc := NewLoginUseCase(userRepo, &stubPasswordVerifier{}, &stubTokenGenerator{})
	in := MustNewLoginInput("alice@dogpaw.com", "correct-password", fixedNowFunc())

	_, err := uc.Execute(context.Background(), in)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_TokenGeneratorFailure(t *testing.T) {
	t.Parallel()

	tokenErr := errors.New("signing key unavailable")
	userRepo := &mockUserRepository{
		getByEmail: func(_ context.Context, _ string) (*domain.User, error) {
			return loginActiveUser(), nil
		},
	}
	tokenGen := &stubTokenGenerator{
		generate: func(_ *domain.User) (string, error) {
			return "", tokenErr
		},
	}
	uc := NewLoginUseCase(userRepo, &stubPasswordVerifier{}, tokenGen)
	in := MustNewLoginInput("alice@dogpaw.com", "correct-password", fixedNowFunc())

	_, err := uc.Execute(context.Background(), in)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, tokenErr), "must wrap original token generator error")
}

func TestLogin_RepositoryError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("connection refused")
	userRepo := &mockUserRepository{
		getByEmail: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, repoErr
		},
	}
	uc := NewLoginUseCase(userRepo, &stubPasswordVerifier{}, &stubTokenGenerator{})
	in := MustNewLoginInput("alice@dogpaw.com", "correct-password", fixedNowFunc())

	_, err := uc.Execute(context.Background(), in)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, repoErr), "must wrap original repository error")
}
