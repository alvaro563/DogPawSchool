package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestNewInvitation(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	expires := now.Add(24 * time.Hour)

	t.Run("happy_path", func(t *testing.T) {
		inv, err := domain.NewInvitation(1, 1, "ana@dogpaw.es", "tok123", domain.RoleRegular, domain.InvitationPending, expires, now, now)
		assert.NoError(t, err)
		assert.NotNil(t, inv)
		assert.Equal(t, 1, inv.ID())
		assert.Equal(t, "ana@dogpaw.es", inv.Email())
		assert.Equal(t, "tok123", inv.Token())
		assert.Equal(t, domain.RoleRegular, inv.Role())
		assert.Equal(t, domain.InvitationPending, inv.Status())
		assert.Equal(t, 1, inv.CreatedBy())
		assert.Equal(t, expires, inv.ExpiresAt())
		assert.Equal(t, now, inv.CreatedAt())
		assert.Equal(t, now, inv.UpdatedAt())
	})

	t.Run("admin_role", func(t *testing.T) {
		inv, err := domain.NewInvitation(2, 5, "bob@test.com", "tok456", domain.RoleAdmin, domain.InvitationPending, expires, now, now)
		assert.NoError(t, err)
		assert.Equal(t, domain.RoleAdmin, inv.Role())
	})

	t.Run("reconstruct_accepted", func(t *testing.T) {
		inv, err := domain.NewInvitation(3, 1, "c@d.com", "tok789", domain.RoleRegular, domain.InvitationAccepted, expires, now, now)
		assert.NoError(t, err)
		assert.Equal(t, domain.InvitationAccepted, inv.Status())
	})

	t.Run("reconstruct_expired", func(t *testing.T) {
		inv, err := domain.NewInvitation(4, 1, "e@f.com", "tok000", domain.RoleAdmin, domain.InvitationExpired, expires, now, now)
		assert.NoError(t, err)
		assert.Equal(t, domain.InvitationExpired, inv.Status())
	})

	t.Run("reconstruct_revoked", func(t *testing.T) {
		inv, err := domain.NewInvitation(5, 1, "g@h.com", "tok111", domain.RoleRegular, domain.InvitationRevoked, expires, now, now)
		assert.NoError(t, err)
		assert.Equal(t, domain.InvitationRevoked, inv.Status())
	})

	t.Run("invalid_status", func(t *testing.T) {
		_, err := domain.NewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationStatus("UNKNOWN"), expires, now, now)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("validation_errors", func(t *testing.T) {
		tests := []struct {
			name      string
			id        int
			createdBy int
			email     string
			token     string
			role      domain.UserRole
			status    domain.InvitationStatus
			expiresAt time.Time
			createdAt time.Time
			updatedAt time.Time
			wantInErr string
		}{
			{
				"negative_id",
				-1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now,
				"id must not be negative",
			},
			{
				"zero_created_by",
				0, 0, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now,
				"createdBy must be greater than 0",
			},
			{
				"negative_created_by",
				0, -1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now,
				"createdBy must be greater than 0",
			},
			{
				"empty_email",
				0, 1, "", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now,
				"email must not be empty",
			},
			{
				"malformed_email",
				0, 1, "not-an-email", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now,
				"invalid email format",
			},
			{
				"empty_token",
				0, 1, "a@b.com", "", domain.RoleRegular, domain.InvitationPending, expires, now, now,
				"token must not be empty",
			},
			{
				"invalid_role",
				0, 1, "a@b.com", "t", domain.UserRole("SUPER"), domain.InvitationPending, expires, now, now,
				"invalid role",
			},
			{
				"zero_created_at",
				0, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, time.Time{}, now,
				"createdAt must be a valid time",
			},
			{
				"zero_updated_at",
				0, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, now, time.Time{},
				"updatedAt must be a valid time",
			},
			{
				"zero_expires_at",
				0, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, time.Time{}, now, now,
				"expiresAt must be a valid time",
			},
			{
				"expires_at_before_created_at",
				0, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, now.Add(-time.Hour), now, now,
				"expiresAt must be after createdAt",
			},
			{
				"expires_at_equals_created_at",
				0, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, now, now, now,
				"expiresAt must be after createdAt",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewInvitation(
					tt.id, tt.createdBy, tt.email, tt.token, tt.role, tt.status,
					tt.expiresAt, tt.createdAt, tt.updatedAt,
				)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantInErr)
			})
		}
	})
}

func TestNewPendingInvitation(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)

	t.Run("happy_path", func(t *testing.T) {
		inv, err := domain.NewPendingInvitation(1, "ana@dogpaw.es", "tok123", domain.RoleRegular, expires, now)
		assert.NoError(t, err)
		assert.NotNil(t, inv)
		assert.Equal(t, 0, inv.ID(), "id should be 0 for new invitations")
		assert.Equal(t, "ana@dogpaw.es", inv.Email())
		assert.Equal(t, "tok123", inv.Token())
		assert.Equal(t, domain.RoleRegular, inv.Role())
		assert.Equal(t, domain.InvitationPending, inv.Status())
		assert.Equal(t, 1, inv.CreatedBy())
		assert.Equal(t, expires, inv.ExpiresAt())
		assert.Equal(t, now, inv.CreatedAt())
		assert.Equal(t, now, inv.UpdatedAt())
	})

	t.Run("admin_role", func(t *testing.T) {
		inv, err := domain.NewPendingInvitation(5, "bob@test.com", "tok456", domain.RoleAdmin, expires, now)
		assert.NoError(t, err)
		assert.Equal(t, domain.RoleAdmin, inv.Role())
	})

	t.Run("validation_errors", func(t *testing.T) {
		tests := []struct {
			name      string
			createdBy int
			email     string
			token     string
			role      domain.UserRole
			expiresAt time.Time
			now       time.Time
			wantInErr string
		}{
			{
				"zero_created_by",
				0, "a@b.com", "t", domain.RoleRegular, expires, now,
				"createdBy must be greater than 0",
			},
			{
				"negative_created_by",
				-1, "a@b.com", "t", domain.RoleRegular, expires, now,
				"createdBy must be greater than 0",
			},
			{
				"empty_email",
				1, "", "t", domain.RoleRegular, expires, now,
				"email must not be empty",
			},
			{
				"malformed_email",
				1, "not-an-email", "t", domain.RoleRegular, expires, now,
				"invalid email format",
			},
			{
				"empty_token",
				1, "a@b.com", "", domain.RoleRegular, expires, now,
				"token must not be empty",
			},
			{
				"invalid_role",
				1, "a@b.com", "t", domain.UserRole("SUPER"), expires, now,
				"invalid role",
			},
			{
				"zero_now",
				1, "a@b.com", "t", domain.RoleRegular, expires, time.Time{},
				"now must be a valid time",
			},
			{
				"zero_expires_at",
				1, "a@b.com", "t", domain.RoleRegular, time.Time{}, now,
				"expiresAt must be a valid time",
			},
			{
				"expires_at_before_now",
				1, "a@b.com", "t", domain.RoleRegular, now.Add(-time.Hour), now,
				"expiresAt must be after now",
			},
			{
				"expires_at_equals_now",
				1, "a@b.com", "t", domain.RoleRegular, now, now,
				"expiresAt must be after now",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewPendingInvitation(
					tt.createdBy, tt.email, tt.token, tt.role,
					tt.expiresAt, tt.now,
				)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantInErr)
			})
		}
	})
}

func TestInvitation_IsExpired(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)

	t.Run("not_expired", func(t *testing.T) {
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending,
			now.Add(time.Hour), now, now)
		assert.False(t, inv.IsExpired(now))
	})

	t.Run("expired", func(t *testing.T) {
		past := now.Add(-2 * time.Hour)
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending,
			now.Add(-time.Hour), past, past)
		assert.True(t, inv.IsExpired(now))
	})

	t.Run("boundary_exactly_now", func(t *testing.T) {
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending,
			now.Add(time.Second), now, now)
		assert.False(t, inv.IsExpired(now))
	})
}

func TestInvitation_CanBeUsed(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)

	t.Run("pending_and_not_expired", func(t *testing.T) {
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending,
			now.Add(time.Hour), now, now)
		assert.True(t, inv.CanBeUsed(now))
	})

	t.Run("pending_but_expired", func(t *testing.T) {
		past := now.Add(-2 * time.Hour)
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending,
			now.Add(-time.Hour), past, past)
		assert.False(t, inv.CanBeUsed(now))
	})

	t.Run("accepted_not_expired", func(t *testing.T) {
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending,
			now.Add(time.Hour), now, now)
		_ = inv.Accept()
		assert.False(t, inv.CanBeUsed(now))
	})

	t.Run("revoked_not_expired", func(t *testing.T) {
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending,
			now.Add(time.Hour), now, now)
		_ = inv.Revoke()
		assert.False(t, inv.CanBeUsed(now))
	})
}

func TestInvitation_Accept(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)

	t.Run("happy_path", func(t *testing.T) {
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now)
		err := inv.Accept()
		assert.NoError(t, err)
		assert.Equal(t, domain.InvitationAccepted, inv.Status())
	})

	t.Run("error_when_already_accepted", func(t *testing.T) {
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now)
		_ = inv.Accept()
		err := inv.Accept()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot accept")
	})

	t.Run("error_when_revoked", func(t *testing.T) {
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now)
		_ = inv.Revoke()
		err := inv.Accept()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot accept")
	})
}

func TestInvitation_Revoke(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)

	t.Run("happy_path", func(t *testing.T) {
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now)
		err := inv.Revoke()
		assert.NoError(t, err)
		assert.Equal(t, domain.InvitationRevoked, inv.Status())
	})

	t.Run("error_when_already_accepted", func(t *testing.T) {
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now)
		_ = inv.Accept()
		err := inv.Revoke()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot revoke")
	})

	t.Run("error_when_already_revoked", func(t *testing.T) {
		inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now)
		_ = inv.Revoke()
		err := inv.Revoke()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot revoke")
	})
}

func TestInvitationStatus_IsValid(t *testing.T) {
	t.Parallel()
	assert.True(t, domain.InvitationPending.IsValid())
	assert.True(t, domain.InvitationAccepted.IsValid())
	assert.True(t, domain.InvitationExpired.IsValid())
	assert.True(t, domain.InvitationRevoked.IsValid())
	assert.False(t, domain.InvitationStatus("").IsValid())
	assert.False(t, domain.InvitationStatus("UNKNOWN").IsValid())
}

func TestInvitation_CanBeUsed_StatusTransitions(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)

	inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now)
	assert.True(t, inv.CanBeUsed(now), "should be usable when pending")

	_ = inv.Accept()
	assert.False(t, inv.CanBeUsed(now), "should not be usable after accept")

	inv2 := domain.MustNewInvitation(2, 1, "b@c.com", "t2", domain.RoleAdmin, domain.InvitationPending, expires, now, now)
	_ = inv2.Revoke()
	assert.False(t, inv2.CanBeUsed(now), "should not be usable after revoke")
}

func TestInvitation_EmailValidationError(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)

	_, err := domain.NewInvitation(1, 1, "invalid", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email format")
}

func TestInvitation_AcceptRejectsMultipleTransitions(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)

	inv := domain.MustNewInvitation(1, 1, "a@b.com", "t", domain.RoleRegular, domain.InvitationPending, expires, now, now)

	assert.NoError(t, inv.Accept())
	err := inv.Accept()
	assert.True(t, errors.Is(err, nil) == false)
	assert.Contains(t, err.Error(), "cannot accept")
}

func TestInvitation_ReconstructAllStatuses(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)

	statuses := []domain.InvitationStatus{
		domain.InvitationPending,
		domain.InvitationAccepted,
		domain.InvitationExpired,
		domain.InvitationRevoked,
	}
	for _, s := range statuses {
		t.Run(string(s), func(t *testing.T) {
			inv, err := domain.NewInvitation(10, 1, "x@y.com", "tok", domain.RoleRegular, s, expires, now, now)
			assert.NoError(t, err)
			assert.Equal(t, s, inv.Status())
		})
	}
}
