package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func fixedNow() func() time.Time {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return now }
}

// validPassword returns a 60-char string that satisfies the DB CHECK.
func validPassword() string {
	return strings.Repeat("a", 60)
}

// --- Mocks ---

type mockInvitationRepository struct {
	getByID     func(ctx context.Context, id int) (*domain.Invitation, error)
	getByToken  func(ctx context.Context, token string) (*domain.Invitation, error)
	create      func(ctx context.Context, inv *domain.Invitation) (int, error)
	update      func(ctx context.Context, inv *domain.Invitation) error
	listPending func(ctx context.Context, limit, offset int) ([]*domain.Invitation, error)
	listByEmail func(ctx context.Context, email string) ([]*domain.Invitation, error)
}

func (m *mockInvitationRepository) Create(ctx context.Context, inv *domain.Invitation) (int, error) {
	if m.create != nil {
		return m.create(ctx, inv)
	}
	return 0, nil
}

func (m *mockInvitationRepository) GetByID(ctx context.Context, id int) (*domain.Invitation, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, nil
}

func (m *mockInvitationRepository) GetByToken(ctx context.Context, token string) (*domain.Invitation, error) {
	if m.getByToken != nil {
		return m.getByToken(ctx, token)
	}
	return nil, nil
}

func (m *mockInvitationRepository) Update(ctx context.Context, inv *domain.Invitation) error {
	if m.update != nil {
		return m.update(ctx, inv)
	}
	return nil
}

func (m *mockInvitationRepository) ListPending(ctx context.Context, limit, offset int) ([]*domain.Invitation, error) {
	if m.listPending != nil {
		return m.listPending(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockInvitationRepository) ListByEmail(ctx context.Context, email string) ([]*domain.Invitation, error) {
	if m.listByEmail != nil {
		return m.listByEmail(ctx, email)
	}
	return nil, nil
}

type mockUserRepository struct {
	create       func(ctx context.Context, user *domain.User) error
	update       func(ctx context.Context, user *domain.User) error
	getByID      func(ctx context.Context, id int) (*domain.User, error)
	getByEmail   func(ctx context.Context, email string) (*domain.User, error)
	listAll      func(ctx context.Context) ([]*domain.User, error)
	listAllPaged func(ctx context.Context, limit, offset int) ([]*domain.User, error)
	delete       func(ctx context.Context, id int) error
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if m.create != nil {
		return m.create(ctx, user)
	}
	return nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *domain.User) error {
	if m.update != nil {
		return m.update(ctx, user)
	}
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmail != nil {
		return m.getByEmail(ctx, email)
	}
	return nil, nil
}

func (m *mockUserRepository) ListAll(ctx context.Context) ([]*domain.User, error) {
	if m.listAll != nil {
		return m.listAll(ctx)
	}
	return nil, nil
}

func (m *mockUserRepository) ListAllPaged(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	if m.listAllPaged != nil {
		return m.listAllPaged(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id int) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

type stubTransactor struct {
	fn func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (s *stubTransactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if s.fn != nil {
		return s.fn(ctx, fn)
	}
	return fn(ctx)
}

// --- Helpers ---

func pendingInvitation(now time.Time) *domain.Invitation {
	expires := now.Add(48 * time.Hour)
	inv, err := domain.NewPendingInvitation(1, "ana@dogpaw.es", "valid-token-123", domain.RoleRegular, expires, now)
	if err != nil {
		panic(err)
	}
	return inv
}

func acceptedInvitation(now time.Time) *domain.Invitation {
	inv := pendingInvitation(now)
	_ = inv.Accept()
	return inv
}

func revokedInvitation(now time.Time) *domain.Invitation {
	inv := pendingInvitation(now)
	_ = inv.Revoke()
	return inv
}

func expiredInvitation(now time.Time) *domain.Invitation {
	past := now.Add(-2 * time.Hour)
	expires := now.Add(-1 * time.Hour)
	inv, err := domain.NewPendingInvitation(1, "ana@dogpaw.es", "expired-token", domain.RoleRegular, expires, past)
	if err != nil {
		panic(err)
	}
	return inv
}

// --- Tests ---

func TestNewRegisterWithInvitationInput(t *testing.T) {
	scenarios := []struct {
		name          string
		factory       func() (RegisterWithInvitationInput, error)
		expectedField string
	}{
		{
			"empty_token",
			func() (RegisterWithInvitationInput, error) {
				return NewRegisterWithInvitationInput("", "Ana", validPassword(), fixedNow())
			},
			"token",
		},
		{
			"empty_name",
			func() (RegisterWithInvitationInput, error) {
				return NewRegisterWithInvitationInput("tok", "", validPassword(), fixedNow())
			},
			"name",
		},
		{
			"short_password",
			func() (RegisterWithInvitationInput, error) {
				return NewRegisterWithInvitationInput("tok", "Ana", "short", fixedNow())
			},
			"password",
		},
		{
			"password_exactly_59_chars",
			func() (RegisterWithInvitationInput, error) {
				return NewRegisterWithInvitationInput("tok", "Ana", strings.Repeat("a", 59), fixedNow())
			},
			"password",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			_, err := s.factory()
			assert.Error(t, err)
			var verr *ValidationError
			assert.True(t, errors.As(err, &verr), "expected *ValidationError, got %T", err)
			assert.Equal(t, s.expectedField, verr.Field)
		})
	}
}

func TestNewRegisterWithInvitationInput_HappyPath(t *testing.T) {
	now := fixedNow()
	in, err := NewRegisterWithInvitationInput("tok", "Ana", validPassword(), now)
	require.NoError(t, err)
	assert.Equal(t, "tok", in.Token())
	assert.Equal(t, "Ana", in.Name())
	assert.Equal(t, validPassword(), in.Password())
	assert.Equal(t, now(), in.Now())
}

func TestNewRegisterWithInvitationInput_NilNowDefaults(t *testing.T) {
	in, err := NewRegisterWithInvitationInput("tok", "Ana", validPassword(), nil)
	require.NoError(t, err)
	assert.False(t, in.Now().IsZero(), "now should default to time.Now")
}

func TestRegisterWithInvitationUseCase_Execute(t *testing.T) {
	fixed := fixedNow()

	t.Run("happy_path", func(t *testing.T) {
		inv := pendingInvitation(fixed())
		var capturedUser *domain.User
		var capturedInv *domain.Invitation

		invRepo := &mockInvitationRepository{
			getByToken: func(_ context.Context, token string) (*domain.Invitation, error) {
				assert.Equal(t, "valid-token-123", token)
				return inv, nil
			},
			update: func(_ context.Context, i *domain.Invitation) error {
				capturedInv = i
				return nil
			},
		}
		userRepo := &mockUserRepository{
			create: func(_ context.Context, u *domain.User) error {
				capturedUser = u
				return nil
			},
		}
		uc := NewRegisterWithInvitationUseCase(&stubTransactor{}, invRepo, userRepo)
		in := MustNewRegisterWithInvitationInput("valid-token-123", "Ana Such", validPassword(), fixed)

		out, err := uc.Execute(context.Background(), in)

		require.NoError(t, err)
		require.NotNil(t, out.User)
		assert.Equal(t, "Ana Such", out.User.Name())
		assert.Equal(t, "ana@dogpaw.es", out.User.Email())
		assert.Equal(t, domain.RoleRegular, out.User.Role())
		assert.True(t, out.User.IsActive())
		require.NotNil(t, capturedUser)
		assert.NotEqual(t, validPassword(), capturedUser.Password(), "password should be hashed")
		require.NotNil(t, capturedInv)
		assert.Equal(t, domain.InvitationAccepted, capturedInv.Status())
	})

	t.Run("token_not_found", func(t *testing.T) {
		invRepo := &mockInvitationRepository{
			getByToken: func(_ context.Context, _ string) (*domain.Invitation, error) {
				return nil, domain.ErrNotFound
			},
		}
		uc := NewRegisterWithInvitationUseCase(&stubTransactor{}, invRepo, &mockUserRepository{})
		in := MustNewRegisterWithInvitationInput("nonexistent", "Ana", validPassword(), fixed)

		_, err := uc.Execute(context.Background(), in)

		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("invitation_expired", func(t *testing.T) {
		inv := expiredInvitation(fixed())
		invRepo := &mockInvitationRepository{
			getByToken: func(_ context.Context, _ string) (*domain.Invitation, error) {
				return inv, nil
			},
		}
		uc := NewRegisterWithInvitationUseCase(&stubTransactor{}, invRepo, &mockUserRepository{})
		in := MustNewRegisterWithInvitationInput("expired-token", "Ana", validPassword(), fixed)

		_, err := uc.Execute(context.Background(), in)

		assert.ErrorIs(t, err, domain.ErrInvitationInvalid)
	})

	t.Run("invitation_already_accepted", func(t *testing.T) {
		inv := acceptedInvitation(fixed())
		invRepo := &mockInvitationRepository{
			getByToken: func(_ context.Context, _ string) (*domain.Invitation, error) {
				return inv, nil
			},
		}
		uc := NewRegisterWithInvitationUseCase(&stubTransactor{}, invRepo, &mockUserRepository{})
		in := MustNewRegisterWithInvitationInput("tok", "Ana", validPassword(), fixed)

		_, err := uc.Execute(context.Background(), in)

		assert.ErrorIs(t, err, domain.ErrInvitationInvalid)
	})

	t.Run("invitation_already_revoked", func(t *testing.T) {
		inv := revokedInvitation(fixed())
		invRepo := &mockInvitationRepository{
			getByToken: func(_ context.Context, _ string) (*domain.Invitation, error) {
				return inv, nil
			},
		}
		uc := NewRegisterWithInvitationUseCase(&stubTransactor{}, invRepo, &mockUserRepository{})
		in := MustNewRegisterWithInvitationInput("tok", "Ana", validPassword(), fixed)

		_, err := uc.Execute(context.Background(), in)

		assert.ErrorIs(t, err, domain.ErrInvitationInvalid)
	})

	t.Run("user_repo_create_error", func(t *testing.T) {
		repoErr := errors.New("duplicate email")
		inv := pendingInvitation(fixed())
		invRepo := &mockInvitationRepository{
			getByToken: func(_ context.Context, _ string) (*domain.Invitation, error) {
				return inv, nil
			},
		}
		userRepo := &mockUserRepository{
			create: func(_ context.Context, _ *domain.User) error {
				return repoErr
			},
		}
		uc := NewRegisterWithInvitationUseCase(&stubTransactor{}, invRepo, userRepo)
		in := MustNewRegisterWithInvitationInput("tok", "Ana", validPassword(), fixed)

		_, err := uc.Execute(context.Background(), in)

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr), "expected wrapped repoErr")
		// Invitation should NOT be accepted since the transaction rolled back
		assert.Equal(t, domain.InvitationPending, inv.Status())
	})

	t.Run("password_is_bcrypt_hash", func(t *testing.T) {
		var capturedUser *domain.User
		inv := pendingInvitation(fixed())
		invRepo := &mockInvitationRepository{
			getByToken: func(_ context.Context, _ string) (*domain.Invitation, error) {
				return inv, nil
			},
			update: func(_ context.Context, _ *domain.Invitation) error { return nil },
		}
		userRepo := &mockUserRepository{
			create: func(_ context.Context, u *domain.User) error {
				capturedUser = u
				return nil
			},
		}
		uc := NewRegisterWithInvitationUseCase(&stubTransactor{}, invRepo, userRepo)
		in := MustNewRegisterWithInvitationInput("tok", "Ana", validPassword(), fixed)

		_, err := uc.Execute(context.Background(), in)

		require.NoError(t, err)
		require.NotNil(t, capturedUser)
		assert.True(t, strings.HasPrefix(capturedUser.Password(), "$2a$"), "password should start with bcrypt prefix")
		assert.Len(t, capturedUser.Password(), 60, "bcrypt hash is 60 chars")
	})

	t.Run("transaction_rolls_back_on_user_create_failure", func(t *testing.T) {
		inv := pendingInvitation(fixed())
		txCommitted := false
		invRepo := &mockInvitationRepository{
			getByToken: func(_ context.Context, _ string) (*domain.Invitation, error) {
				return inv, nil
			},
			update: func(_ context.Context, _ *domain.Invitation) error { return nil },
		}
		userRepo := &mockUserRepository{
			create: func(_ context.Context, _ *domain.User) error {
				return errors.New("db constraint violation")
			},
		}
		transactor := &stubTransactor{
			fn: func(ctx context.Context, fn func(ctx context.Context) error) error {
				err := fn(ctx)
				if err != nil {
					// Simulate rollback — don't commit
					return err
				}
				txCommitted = true
				return nil
			},
		}
		uc := NewRegisterWithInvitationUseCase(transactor, invRepo, userRepo)
		in := MustNewRegisterWithInvitationInput("tok", "Ana", validPassword(), fixed)

		_, err := uc.Execute(context.Background(), in)

		assert.Error(t, err)
		assert.False(t, txCommitted, "transaction should not commit on error")
		assert.Equal(t, domain.InvitationPending, inv.Status(), "invitation should remain PENDING after rollback")
	})
}
