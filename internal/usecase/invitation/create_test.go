package invitation

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

var hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

func fixedNow() func() time.Time {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return now }
}

func TestNewCreateInvitationInput(t *testing.T) {
	scenarios := []struct {
		name          string
		factory       func() (CreateInvitationInput, error)
		expectedField string
	}{
		{
			"zero_created_by",
			func() (CreateInvitationInput, error) {
				return NewCreateInvitationInput(0, "a@b.com", domain.RoleRegular, fixedNow())
			},
			"created_by",
		},
		{
			"negative_created_by",
			func() (CreateInvitationInput, error) {
				return NewCreateInvitationInput(-1, "a@b.com", domain.RoleRegular, fixedNow())
			},
			"created_by",
		},
		{
			"empty_email",
			func() (CreateInvitationInput, error) {
				return NewCreateInvitationInput(1, "", domain.RoleRegular, fixedNow())
			},
			"email",
		},
		{
			"malformed_email_no_at",
			func() (CreateInvitationInput, error) {
				return NewCreateInvitationInput(1, "notanemail", domain.RoleRegular, fixedNow())
			},
			"email",
		},
		{
			"malformed_email_no_domain",
			func() (CreateInvitationInput, error) {
				return NewCreateInvitationInput(1, "user@", domain.RoleRegular, fixedNow())
			},
			"email",
		},
		{
			"invalid_role",
			func() (CreateInvitationInput, error) {
				return NewCreateInvitationInput(1, "a@b.com", domain.UserRole("SUPER"), fixedNow())
			},
			"role",
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

func TestNewCreateInvitationInput_NilNowDefaults(t *testing.T) {
	in, err := NewCreateInvitationInput(1, "a@b.com", domain.RoleRegular, nil)
	require.NoError(t, err)
	assert.False(t, in.Now().IsZero(), "now should default to time.Now")
}

func TestNewCreateInvitationInput_HappyPath(t *testing.T) {
	now := fixedNow()
	in, err := NewCreateInvitationInput(1, "ana@dogpaw.es", domain.RoleAdmin, now)
	require.NoError(t, err)
	assert.Equal(t, 1, in.CreatedBy())
	assert.Equal(t, "ana@dogpaw.es", in.Email())
	assert.Equal(t, domain.RoleAdmin, in.Role())
	assert.Equal(t, now(), in.Now())
}

func TestCreateInvitationUseCase_Execute(t *testing.T) {
	fixed := fixedNow()

	t.Run("happy_path", func(t *testing.T) {
		var capturedInv *domain.Invitation
		mock := &mockInvitationRepository{
			create: func(_ context.Context, inv *domain.Invitation) (int, error) {
				capturedInv = inv
				return 42, nil
			},
		}
		uc := NewCreateInvitationUseCase(mock)
		in := MustNewCreateInvitationInput(1, "ana@dogpaw.es", domain.RoleRegular, fixed)

		out, err := uc.Execute(context.Background(), in)

		require.NoError(t, err)
		assert.Equal(t, 42, out.ID)
		assert.NotEmpty(t, out.Token)
		assert.Regexp(t, hex64, out.Token, "token should be 64 hex chars")
		require.NotNil(t, capturedInv)
		assert.Equal(t, "ana@dogpaw.es", capturedInv.Email())
		assert.Equal(t, domain.RoleRegular, capturedInv.Role())
		assert.Equal(t, domain.InvitationPending, capturedInv.Status())
		assert.Equal(t, 0, capturedInv.ID(), "id should be 0 before persist")
		assert.Equal(t, 1, capturedInv.CreatedBy())
	})

	t.Run("token_is_64_hex_chars", func(t *testing.T) {
		mock := &mockInvitationRepository{
			create: func(_ context.Context, _ *domain.Invitation) (int, error) {
				return 1, nil
			},
		}
		uc := NewCreateInvitationUseCase(mock)
		in := MustNewCreateInvitationInput(1, "b@c.com", domain.RoleAdmin, fixed)

		out, err := uc.Execute(context.Background(), in)

		require.NoError(t, err)
		assert.Len(t, out.Token, 64, "32 random bytes = 64 hex chars")
		assert.Regexp(t, `^[0-9a-f]{64}$`, out.Token)
	})

	t.Run("expires_at_is_48h_from_now", func(t *testing.T) {
		now := fixed()
		var capturedInv *domain.Invitation
		mock := &mockInvitationRepository{
			create: func(_ context.Context, inv *domain.Invitation) (int, error) {
				capturedInv = inv
				return 1, nil
			},
		}
		uc := NewCreateInvitationUseCase(mock)
		in := MustNewCreateInvitationInput(1, "c@d.com", domain.RoleRegular, fixed)

		_, err := uc.Execute(context.Background(), in)

		require.NoError(t, err)
		require.NotNil(t, capturedInv)
		expectedExpiry := now.Add(48 * time.Hour)
		assert.Equal(t, expectedExpiry, capturedInv.ExpiresAt())
	})

	t.Run("repo_create_error_propagated", func(t *testing.T) {
		repoErr := errors.New("database connection lost")
		mock := &mockInvitationRepository{
			create: func(_ context.Context, _ *domain.Invitation) (int, error) {
				return 0, repoErr
			},
		}
		uc := NewCreateInvitationUseCase(mock)
		in := MustNewCreateInvitationInput(1, "d@e.com", domain.RoleRegular, fixed)

		_, err := uc.Execute(context.Background(), in)

		assert.Error(t, err)
		assert.True(t, errors.Is(err, repoErr), "expected wrapped repoErr")
	})

	t.Run("invitation_status_is_pending", func(t *testing.T) {
		var capturedInv *domain.Invitation
		mock := &mockInvitationRepository{
			create: func(_ context.Context, inv *domain.Invitation) (int, error) {
				capturedInv = inv
				return 1, nil
			},
		}
		uc := NewCreateInvitationUseCase(mock)
		in := MustNewCreateInvitationInput(1, "f@g.com", domain.RoleAdmin, fixed)

		_, err := uc.Execute(context.Background(), in)

		require.NoError(t, err)
		require.NotNil(t, capturedInv)
		assert.Equal(t, domain.InvitationPending, capturedInv.Status())
	})

	t.Run("each_invocation_produces_different_token", func(t *testing.T) {
		mock := &mockInvitationRepository{
			create: func(_ context.Context, _ *domain.Invitation) (int, error) {
				return 1, nil
			},
		}
		uc := NewCreateInvitationUseCase(mock)
		in := MustNewCreateInvitationInput(1, "h@i.com", domain.RoleRegular, fixed)

		out1, _ := uc.Execute(context.Background(), in)
		out2, _ := uc.Execute(context.Background(), in)

		assert.NotEqual(t, out1.Token, out2.Token, "tokens should be unique")
	})
}
