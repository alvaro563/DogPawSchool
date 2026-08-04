package domain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dogpaw/internal/domain"
)

func TestNewUser(t *testing.T) {
	t.Parallel()
	t.Run("happy_path", func(t *testing.T) {
		u, err := domain.NewUser(1, "Ana Such", "ana@dogpaw.es", "hashed_pw", domain.RoleAdmin)
		assert.NoError(t, err)
		assert.NotNil(t, u)
		assert.Equal(t, 1, u.ID())
		assert.Equal(t, "Ana Such", u.Name())
		assert.True(t, u.IsActive())
		assert.True(t, u.IsAdmin())
	})

	t.Run("validation_errors", func(t *testing.T) {
		tests := []struct {
			name      string
			id        int
			n         string
			email     string
			pw        string
			role      domain.UserRole
			wantInErr string
		}{
			{"negative_id", -1, "n", "e", "p", domain.RoleAdmin, "id must not be negative"},
			{"empty_name", 1, "", "e", "p", domain.RoleAdmin, "name must not be empty"},
			{"empty_email", 1, "n", "", "p", domain.RoleAdmin, "email must not be empty"},
			{"empty_password", 1, "n", "e@t.com", "", domain.RoleAdmin, "password must not be empty"},
			{"invalid_role", 1, "n", "e@t.com", "p", domain.UserRole("SUPER"), "invalid role"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewUser(tt.id, tt.n, tt.email, tt.pw, tt.role)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantInErr)
			})
		}
	})
}

func TestUser_IsAdmin(t *testing.T) {
	t.Parallel()
	admin, _ := domain.NewUser(1, "A", "a@b.com", "p", domain.RoleAdmin)
	regular, _ := domain.NewUser(2, "B", "b@b.com", "p", domain.RoleRegular)
	assert.True(t, admin.IsAdmin())
	assert.False(t, regular.IsAdmin())
}

func TestUser_CanLogin(t *testing.T) {
	t.Parallel()
	u, _ := domain.NewUser(1, "A", "a@b.com", "p", domain.RoleAdmin)
	assert.True(t, u.CanLogin())
	u.Deactivate()
	assert.False(t, u.CanLogin())
	u.Activate()
	assert.True(t, u.CanLogin())
}

func TestUser_Activate_Deactivate(t *testing.T) {
	t.Parallel()
	u, _ := domain.NewUser(1, "A", "a@b.com", "p", domain.RoleAdmin)
	u.Deactivate()
	assert.False(t, u.IsActive())
	u.Activate()
	assert.True(t, u.IsActive())
}

func TestUserRole_IsValid(t *testing.T) {
	t.Parallel()
	assert.True(t, domain.RoleAdmin.IsValid())
	assert.True(t, domain.RoleRegular.IsValid())
	assert.False(t, domain.UserRole("").IsValid())
	assert.False(t, domain.UserRole("SUPER").IsValid())
}

func TestUser_ApplyPatch(t *testing.T) {
	t.Parallel()
	t.Run("empty_patch_is_noop", func(t *testing.T) {
		u, err := domain.NewUser(1, "Ana", "ana@dogpaw.es", "hashed_pw_60chars_xxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleAdmin)
		require.NoError(t, err)
		err = u.ApplyPatch(domain.UserPatch{})
		assert.NoError(t, err)
		assert.Equal(t, "Ana", u.Name())
		assert.Equal(t, "ana@dogpaw.es", u.Email())
		assert.Equal(t, domain.RoleAdmin, u.Role(), "role preserved")
		assert.True(t, u.IsActive(), "is_active preserved")
	})

	t.Run("name_only_update_preserves_email", func(t *testing.T) {
		u, _ := domain.NewUser(1, "Ana", "ana@dogpaw.es", "hashed_pw_60chars_xxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleAdmin)
		newName := "Ana Such"
		err := u.ApplyPatch(domain.UserPatch{Name: &newName})
		assert.NoError(t, err)
		assert.Equal(t, "Ana Such", u.Name())
		assert.Equal(t, "ana@dogpaw.es", u.Email(), "email preserved")
	})

	t.Run("email_only_update_preserves_name", func(t *testing.T) {
		u, _ := domain.NewUser(1, "Ana", "ana@dogpaw.es", "hashed_pw_60chars_xxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleAdmin)
		newEmail := "ana.such@dogpaw.es"
		err := u.ApplyPatch(domain.UserPatch{Email: &newEmail})
		assert.NoError(t, err)
		assert.Equal(t, "Ana", u.Name(), "name preserved")
		assert.Equal(t, "ana.such@dogpaw.es", u.Email())
	})

	t.Run("name_and_email_updated", func(t *testing.T) {
		u, _ := domain.NewUser(1, "Ana", "ana@dogpaw.es", "hashed_pw_60chars_xxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleAdmin)
		newName := "Luna"
		newEmail := "luna@dogpaw.es"
		err := u.ApplyPatch(domain.UserPatch{Name: &newName, Email: &newEmail})
		assert.NoError(t, err)
		assert.Equal(t, "Luna", u.Name())
		assert.Equal(t, "luna@dogpaw.es", u.Email())
	})

	t.Run("name_is_trimmed", func(t *testing.T) {
		u, _ := domain.NewUser(1, "Ana", "ana@dogpaw.es", "hashed_pw_60chars_xxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleAdmin)
		padded := "  Ana Such  "
		err := u.ApplyPatch(domain.UserPatch{Name: &padded})
		assert.NoError(t, err)
		assert.Equal(t, "Ana Such", u.Name())
	})

	t.Run("validation_errors", func(t *testing.T) {
		tests := []struct {
			name      string
			patch     domain.UserPatch
			wantField string
		}{
			{"empty_name", domain.UserPatch{Name: stringPtr("")}, "name"},
			{"whitespace_only_name", domain.UserPatch{Name: stringPtr("   ")}, "name"},
			{"empty_email", domain.UserPatch{Email: stringPtr("")}, "email"},
			{"whitespace_only_email", domain.UserPatch{Email: stringPtr("   ")}, "email"},
			{"malformed_email_no_at", domain.UserPatch{Email: stringPtr("anadogpaw.es")}, "email"},
			{"malformed_email_no_domain_dot", domain.UserPatch{Email: stringPtr("ana@dogpaw")}, "email"},
			{"malformed_email_with_space", domain.UserPatch{Email: stringPtr("ana @dogpaw.es")}, "email"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				u, _ := domain.NewUser(1, "Ana", "ana@dogpaw.es", "hashed_pw_60chars_xxxxxxxxxxxxxxxxxxxxxxxxxxxx", domain.RoleAdmin)
				err := u.ApplyPatch(tt.patch)
				assert.Error(t, err)
				var verr *domain.UserValidationError
				assert.True(t, errors.As(err, &verr), "expected *UserValidationError, got %T", err)
				if errors.As(err, &verr) {
					assert.Equal(t, tt.wantField, verr.Field)
				}
			})
		}
	})
}

func TestUserPatch_IsEmpty(t *testing.T) {
	t.Parallel()
	assert.True(t, domain.UserPatch{}.IsEmpty())
	assert.True(t, (domain.UserPatch{}).IsEmpty())
	name := "x"
	assert.False(t, domain.UserPatch{Name: &name}.IsEmpty())
	email := "x@y.z"
	assert.False(t, domain.UserPatch{Email: &email}.IsEmpty())
}

func stringPtr(s string) *string { return &s }
