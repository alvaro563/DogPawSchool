package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"dogpaw/internal/domain"
)

func TestNewUser(t *testing.T) {
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
			{"empty_password", 1, "n", "e", "", domain.RoleAdmin, "password must not be empty"},
			{"invalid_role", 1, "n", "e", "p", domain.UserRole("SUPER"), "invalid role"},
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
	admin, _ := domain.NewUser(1, "A", "a@a", "p", domain.RoleAdmin)
	regular, _ := domain.NewUser(2, "B", "b@b", "p", domain.RoleRegular)
	assert.True(t, admin.IsAdmin())
	assert.False(t, regular.IsAdmin())
}

func TestUser_CanLogin(t *testing.T) {
	u, _ := domain.NewUser(1, "A", "a@a", "p", domain.RoleAdmin)
	assert.True(t, u.CanLogin())
	u.Deactivate()
	assert.False(t, u.CanLogin())
	u.Activate()
	assert.True(t, u.CanLogin())
}

func TestUser_Activate_Deactivate(t *testing.T) {
	u, _ := domain.NewUser(1, "A", "a@a", "p", domain.RoleAdmin)
	u.Deactivate()
	assert.False(t, u.IsActive())
	u.Activate()
	assert.True(t, u.IsActive())
}

func TestUserRole_IsValid(t *testing.T) {
	assert.True(t, domain.RoleAdmin.IsValid())
	assert.True(t, domain.RoleRegular.IsValid())
	assert.False(t, domain.UserRole("").IsValid())
	assert.False(t, domain.UserRole("SUPER").IsValid())
}
