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

// noopAccountLimiter is the test double for domain.AccountLoginLimiter.
// The default behaviour (always allow, never error) matches the
// pre-lockout semantics; tests that exercise the lockout path use
// the explicit check/record fields.
type noopAccountLimiter struct {
	check func(ctx context.Context, email, remoteIP string) error
	record func(ctx context.Context, email, remoteIP string, success bool) error
	reset  func(ctx context.Context, email string) error
}

func (n *noopAccountLimiter) Check(ctx context.Context, email, remoteIP string) error {
	if n.check != nil {
		return n.check(ctx, email, remoteIP)
	}
	return nil
}

func (n *noopAccountLimiter) Record(ctx context.Context, email, remoteIP string, success bool) error {
	if n.record != nil {
		return n.record(ctx, email, remoteIP, success)
	}
	return nil
}

func (n *noopAccountLimiter) ResetByEmail(ctx context.Context, email string) error {
	if n.reset != nil {
		return n.reset(ctx, email)
	}
	return nil
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
				return NewLoginInput("", "secret", "", fixedNowFunc())
			},
			expectedField: "email",
		},
		{
			name: "empty_password",
			factory: func() (LoginInput, error) {
				return NewLoginInput("alice@dogpaw.com", "", "", fixedNowFunc())
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
		MustNewLoginInput("", "pw", "", fixedNowFunc())
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

	uc := NewLoginUseCase(userRepo, verifier, tokenGen, &noopAccountLimiter{})
	in := MustNewLoginInput("alice@dogpaw.com", "correct-password", "", fixedNowFunc())

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
	uc := NewLoginUseCase(userRepo, &stubPasswordVerifier{}, &stubTokenGenerator{}, &noopAccountLimiter{})
	in := MustNewLoginInput("unknown@dogpaw.com", "any-password", "", fixedNowFunc())

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
	uc := NewLoginUseCase(userRepo, verifier, &stubTokenGenerator{}, &noopAccountLimiter{})
	in := MustNewLoginInput("alice@dogpaw.com", "wrong-password", "", fixedNowFunc())

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
	uc := NewLoginUseCase(userRepo, &stubPasswordVerifier{}, &stubTokenGenerator{}, &noopAccountLimiter{})
	in := MustNewLoginInput("alice@dogpaw.com", "correct-password", "", fixedNowFunc())

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
	uc := NewLoginUseCase(userRepo, &stubPasswordVerifier{}, tokenGen, &noopAccountLimiter{})
	in := MustNewLoginInput("alice@dogpaw.com", "correct-password", "", fixedNowFunc())

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
	uc := NewLoginUseCase(userRepo, &stubPasswordVerifier{}, &stubTokenGenerator{}, &noopAccountLimiter{})
	in := MustNewLoginInput("alice@dogpaw.com", "correct-password", "", fixedNowFunc())

	_, err := uc.Execute(context.Background(), in)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, repoErr), "must wrap original repository error")
}

// --- Lockout integration tests (against stub limiter) ---

func TestLogin_LockoutRejectsBeforeBcrypt(t *testing.T) {
	t.Parallel()

	// Lockout rejects BEFORE the password check. We confirm this
	// by providing a verifier that records whether it was called:
	// it must NOT be called.
	verifierCalled := false
	verifier := &stubPasswordVerifier{
		verify: func(_, _ string) error {
			verifierCalled = true
			return nil
		},
	}
	limiter := &noopAccountLimiter{
		check: func(_ context.Context, _, _ string) error {
			return &domain.ErrAccountLockout{
				RetryAfter: 5 * time.Minute,
				Reason:     domain.LockoutReasonEmail,
			}
		},
	}
	uc := NewLoginUseCase(&mockUserRepository{}, verifier, &stubTokenGenerator{}, limiter)
	in := MustNewLoginInput("alice@dogpaw.com", "any-password", "", fixedNowFunc())

	_, err := uc.Execute(context.Background(), in)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrAccountLocked), "lockout sentinel must propagate")
	assert.False(t, verifierCalled, "lockout must short-circuit BEFORE bcrypt verify")

	var lockoutErr *domain.ErrAccountLockout
	require.True(t, errors.As(err, &lockoutErr))
	assert.Equal(t, domain.LockoutReasonEmail, lockoutErr.Reason)
	assert.Equal(t, 5*time.Minute, lockoutErr.RetryAfter)
}

func TestLogin_RecordOnEveryOutcome(t *testing.T) {
	t.Parallel()

	calls := []struct {
		email, ip string
		success   bool
	}{}
	limiter := &noopAccountLimiter{
		record: func(_ context.Context, email, ip string, success bool) error {
			calls = append(calls, struct {
				email, ip string
				success   bool
			}{email, ip, success})
			return nil
		},
	}

	t.Run("wrong_password_records_failure", func(t *testing.T) {
		calls = nil
		userRepo := &mockUserRepository{
			getByEmail: func(_ context.Context, _ string) (*domain.User, error) {
				return loginActiveUser(), nil
			},
		}
		verifier := &stubPasswordVerifier{
			verify: func(_, _ string) error { return errors.New("mismatch") },
		}
		uc := NewLoginUseCase(userRepo, verifier, &stubTokenGenerator{}, limiter)
		_, _ = uc.Execute(context.Background(),
			MustNewLoginInput("alice@dogpaw.com", "wrong", "10.0.0.99", fixedNowFunc()))
		require.Len(t, calls, 1)
		assert.Equal(t, "alice@dogpaw.com", calls[0].email)
		assert.Equal(t, "10.0.0.99", calls[0].ip)
		assert.False(t, calls[0].success)
	})

	t.Run("success_records_then_resets", func(t *testing.T) {
		calls = nil
		resets := 0
		limiter := &noopAccountLimiter{
			record: func(_ context.Context, email, ip string, success bool) error {
				calls = append(calls, struct {
					email, ip string
					success   bool
				}{email, ip, success})
				return nil
			},
			reset: func(_ context.Context, _ string) error {
				resets++
				return nil
			},
		}
		userRepo := &mockUserRepository{
			getByEmail: func(_ context.Context, _ string) (*domain.User, error) {
				return loginActiveUser(), nil
			},
		}
		uc := NewLoginUseCase(userRepo, &stubPasswordVerifier{}, &stubTokenGenerator{}, limiter)
		_, err := uc.Execute(context.Background(),
			MustNewLoginInput("alice@dogpaw.com", "right", "10.0.0.99", fixedNowFunc()))
		require.NoError(t, err)
		require.Len(t, calls, 1)
		assert.True(t, calls[0].success)
		assert.Equal(t, 1, resets, "successful login must reset the per-email counter")
	})
}
